package main

import "C"

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/pull"
	"k8s.io/client-go/kubernetes/scheme"
)

//export PullFromAirgap
func PullFromAirgap(socket, licenseData, airgapDir, downstream, outputFile, registryHost, registryNamespace, username, password string) {
	go func() {
		var ffiResult *FFIResult

		statusClient, err := connectToStatusServer(socket)
		if err != nil {
			fmt.Printf("failed to connect to status server: %s\n", err)
			return
		}
		defer func() {
			statusClient.end(ffiResult)
		}()

		workspace, err := ioutil.TempDir("", "kots-airgap")
		if err != nil {
			fmt.Printf("failed to create temp dir: %s\n", err)
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}
		defer os.RemoveAll(workspace)

		// releaseDir is the contents of the release tar (yaml, no images)
		releaseDir, err := extractAppRelease(workspace, airgapDir)
		if err != nil {
			fmt.Printf("failed to extract app release: %s\n", err)
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		kotsscheme.AddToScheme(scheme.Scheme)
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode([]byte(licenseData), nil, nil)
		if err != nil {
			fmt.Printf("failed to decode license data: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}
		license := obj.(*kotsv1beta1.License)

		licenseFile, err := ioutil.TempFile("", "kots")
		if err != nil {
			fmt.Printf("failed to create temp file: %s\n", err)
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}
		defer os.Remove(licenseFile.Name())

		if err := ioutil.WriteFile(licenseFile.Name(), []byte(licenseData), 0644); err != nil {
			fmt.Printf("failed to write license to temp file: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		// pull to a tmp dir
		tmpRoot, err := ioutil.TempDir("", "kots")
		if err != nil {
			fmt.Printf("failed to create temp root path: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}
		defer os.RemoveAll(tmpRoot)

		pullOptions := pull.PullOptions{
			Downstreams:         []string{downstream},
			LocalPath:           releaseDir,
			LicenseFile:         licenseFile.Name(),
			ExcludeKotsKinds:    true,
			RootDir:             tmpRoot,
			ExcludeAdminConsole: true,
			RewriteImages:       true,
			RewriteImageOptions: pull.RewriteImageOptions{
				ImageFiles: filepath.Join(airgapDir, "images"),
				Host:       registryHost,
				Namespace:  registryNamespace,
				Username:   username,
				Password:   password,
			},
		}

		if _, err := pull.Pull(fmt.Sprintf("replicated://%s", license.Spec.AppSlug), pullOptions); err != nil {
			fmt.Printf("failed to pull upstream: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		// make an archive
		tarGz := archiver.TarGz{
			Tar: &archiver.Tar{
				ImplicitTopLevelFolder: true,
			},
		}

		paths := []string{
			filepath.Join(tmpRoot, "upstream"),
			filepath.Join(tmpRoot, "base"),
			filepath.Join(tmpRoot, "overlays"),
		}

		if err := tarGz.Archive(paths, outputFile); err != nil {
			fmt.Printf("failed to write archive: %s", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		ffiResult = NewFFIResult(0)
	}()
}

func downloadAirgapAchive(workspace string, airgapURL string) (string, error) {
	resp, err := http.Get(airgapURL)
	if err != nil {
		return "", errors.Wrap(err, "failed to download file")
	}

	destDir := filepath.Join(workspace, "extracted-airgap")
	if err := os.Mkdir(destDir, 0744); err != nil {
		return "", errors.Wrap(err, "failed to create tmp dir")
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.Wrapf(err, "unexpected status code: %v", resp.StatusCode)
	}
	defer resp.Body.Close()

	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", errors.Wrapf(err, "unexpected status code: %v", resp.StatusCode)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return "", errors.Wrap(err, "failed to expand archive")
		}

		if hdr.Typeflag != tar.TypeReg {
			// we don't expat anything but files in our archives...
			continue
		}

		fileName := filepath.Join(destDir, hdr.Name)
		if err := os.MkdirAll(filepath.Dir(fileName), 0744); err != nil {
			return "", errors.Wrapf(err, "failed to create path %q", filepath.Dir(fileName))
		}

		err = func() error { // func so we can defer close files... /shrug
			fileWriter, err := os.Create(fileName)
			if err != nil {
				return errors.Wrapf(err, "failed to create file %q", fileName)
			}
			defer fileWriter.Close()

			if _, err := io.Copy(fileWriter, tarReader); err != nil {
				return errors.Wrapf(err, "failed to write file %q", fileName)
			}
			return nil
		}()
		if err != nil {
			return "", err
		}
	}

	return destDir, nil
}

func extractAppRelease(workspace string, airgapDir string) (string, error) {
	files, err := ioutil.ReadDir(airgapDir)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read airgap dir")
	}

	destDir := filepath.Join(workspace, "extracted-app-release")
	if err := os.Mkdir(destDir, 0744); err != nil {
		return "", errors.Wrap(err, "failed to create tmp dir")
	}

	numExtracted := 0
	for _, file := range files {
		if file.IsDir() { // TODO: support nested dirs?
			continue
		}
		err := extractOneArchive(filepath.Join(airgapDir, file.Name()), destDir)
		if err != nil {
			fmt.Printf("ignoring file %q: %v\n", file.Name(), err)
			continue
		}
		numExtracted++
	}

	if numExtracted == 0 {
		return "", errors.New("no release found in airgap archive")
	}

	return destDir, nil
}

func extractOneArchive(tgzFile string, destDir string) error {
	fileReader, err := os.Open(tgzFile)
	if err != nil {
		return errors.Wrap(err, "failed to open release file")
	}

	gzReader, err := gzip.NewReader(fileReader)
	if err != nil {
		return errors.Wrap(err, "failed to create gzip reader")
	}

	tarReader := tar.NewReader(gzReader)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "failed to read release tar")
		}

		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		err = func() error {
			fileName := filepath.Join(destDir, hdr.Name)

			filePath, _ := filepath.Split(fileName)
			err := os.MkdirAll(filePath, 0755)
			if err != nil {
				return errors.Wrapf(err, "failed to create directory %q", filePath)
			}

			fileWriter, err := os.Create(fileName)
			if err != nil {
				return errors.Wrapf(err, "failed to create file %q", hdr.Name)
			}

			defer fileWriter.Close()

			_, err = io.Copy(fileWriter, tarReader)
			if err != nil {
				return errors.Wrapf(err, "failed to write file %q", hdr.Name)
			}

			return nil
		}()
		if err != nil {
			return err
		}
	}

	return nil
}
