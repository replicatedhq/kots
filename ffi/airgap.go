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
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/pull"
	"k8s.io/client-go/kubernetes/scheme"
)

type ImageRef struct {
	Domain string
	Name   string
	Tag    string
	Digest string
}

//export RewriteAndPushImageName
func RewriteAndPushImageName(imageFile, image, registryHost, registryOrg, username, password string) int {
	imageRef, err := parseImageRef(image)
	if err != nil {
		fmt.Printf("failed to parse image %s: %s\n", image, err)
		return 1
	}
	localImage := imageRefToString(imageRef, registryHost, registryOrg)

	cmdArgs := []string{
		"copy",
		"--dest-tls-verify=false",
	}
	if len(username) > 0 && len(password) > 0 {
		cmdArgs = append(cmdArgs, fmt.Sprintf("--dest-creds=%s:%s", username, password))
	}
	cmdArgs = append(cmdArgs,
		fmt.Sprintf("oci-archive:%s", imageFile),
		fmt.Sprintf("docker://%s", localImage),
	)

	cmd := exec.Command("skopeo", cmdArgs...)
	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("failed to run skopeo: %v\n", err)
		if len(cmdOutput) > 0 {
			fmt.Printf("run failed with output: %s\n", cmdOutput)
		}
		return 1
	}

	return 0
}

func parseImageRef(image string) (*ImageRef, error) {
	ref := &ImageRef{}

	// named, err := reference.ParseNormalizedNamed(image)
	parsed, err := reference.ParseAnyReference(image)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse image name %q", image)
	}

	if named, ok := parsed.(reference.Named); ok {
		ref.Domain = reference.Domain(named)
		ref.Name = named.Name()
	} else {
		return nil, errors.New(fmt.Sprintf("unsupported ref type: %T", parsed))
	}

	if tagged, ok := parsed.(reference.Tagged); ok {
		ref.Tag = tagged.Tag()
	} else if can, ok := parsed.(reference.Canonical); ok {
		ref.Digest = can.Digest().String()
	} else {
		ref.Tag = "latest"
	}

	return ref, nil
}

func imageRefToString(ref *ImageRef, registryHost, registryOrg string) string {
	pathParts := strings.Split(ref.Name, "/")
	imageName := fmt.Sprintf("%s/%s", registryOrg, pathParts[len(pathParts)-1])

	// there might be a way to do this with reference package too
	if ref.Digest != "" {
		return fmt.Sprintf("%s/%s@sha256:%s", registryHost, imageName, ref.Digest)
	}
	return fmt.Sprintf("%s/%s:%s", registryHost, imageName, ref.Tag)
}

//export PullFromAirgap
func PullFromAirgap(licenseData, airgapDir, downstream, outputFile, registryHost, registryNamesapce string) int {
	workspace, err := ioutil.TempDir("", "kots-airgap")
	if err != nil {
		fmt.Printf("failed to create temp dir: %s\n", err)
		return 1
	}
	defer os.RemoveAll(workspace)

	// releaseDir is the contents of the release tar (yaml, no images)
	releaseDir, err := extractAppRelease(workspace, airgapDir)
	if err != nil {
		fmt.Printf("failed to extract app release: %s\n", err)
		return 1
	}

	kotsscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(licenseData), nil, nil)
	if err != nil {
		fmt.Printf("failed to decode license data: %s\n", err.Error())
		return 1
	}
	license := obj.(*kotsv1beta1.License)

	licenseFile, err := ioutil.TempFile("", "kots")
	if err != nil {
		fmt.Printf("failed to create temp file: %s\n", err)
		return 1
	}
	defer os.Remove(licenseFile.Name())

	if err := ioutil.WriteFile(licenseFile.Name(), []byte(licenseData), 0644); err != nil {
		fmt.Printf("failed to write license to temp file: %s\n", err.Error())
		return 1
	}

	// pull to a tmp dir
	tmpRoot, err := ioutil.TempDir("", "kots")
	if err != nil {
		fmt.Printf("failed to create temp root path: %s\n", err.Error())
		return 1
	}
	defer os.RemoveAll(tmpRoot)

	pullOptions := pull.PullOptions{
		Downstreams:         []string{downstream},
		LocalPath:           releaseDir,
		LicenseFile:         licenseFile.Name(),
		ExcludeKotsKinds:    true,
		RootDir:             tmpRoot,
		ExcludeAdminConsole: true,
		RewriteImages: pull.RewriteImages{
			ImageFiles: filepath.Join(airgapDir, "images"),
			Host:       registryHost,
			Namespace:  registryNamesapce,
		},
	}

	if _, err := pull.Pull(fmt.Sprintf("replicated://%s", license.Spec.AppSlug), pullOptions); err != nil {
		fmt.Printf("failed to pull upstream: %s\n", err.Error())
		return 1
	}

	// make an archive
	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: true,
		},
	}

	// TODO: Rewrite images....

	paths := []string{
		filepath.Join(tmpRoot, "upstream"),
		filepath.Join(tmpRoot, "base"),
		filepath.Join(tmpRoot, "overlays"),
	}

	if err := tarGz.Archive(paths, outputFile); err != nil {
		fmt.Printf("failed to write archive: %s", err.Error())
		return 1
	}

	return 0
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
			fmt.Printf("ignoring file %q\n", file.Name())
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
