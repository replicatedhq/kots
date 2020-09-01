package airgap

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/airgap/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/preflight"
	"github.com/replicatedhq/kots/kotsadm/pkg/registry"
	registrytypes "github.com/replicatedhq/kots/kotsadm/pkg/registry/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/pull"
	"k8s.io/client-go/kubernetes/scheme"
)

// CreateAppFromAirgap does a lot. Maybe too much. Definitely too much.
// This function assumes that there's an app in the database that doesn't have a version
// After execution, there will be a sequence 0 of the app, and all clusters in the database
// will also have a version
func CreateAppFromAirgap(pendingApp *types.PendingApp, airgapBundle string, registryHost string, namespace string, username string, password string) (finalError error) {
	if err := store.GetStore().SetTaskStatus("airgap-install", "Processing package...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	finishedCh := make(chan struct{})
	defer close(finishedCh)
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				if err := store.GetStore().UpdateTaskStatusTimestamp("airgap-install"); err != nil {
					logger.Error(err)
				}
			case <-finishedCh:
				return
			}
		}
	}()

	defer func() {
		if finalError == nil {
			if err := store.GetStore().ClearTaskStatus("airgap-install"); err != nil {
				logger.Error(errors.Wrap(err, "failed to clear install task status"))
			}
			if err := store.GetStore().SetAppInstallState(pendingApp.ID, "installed"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set app status to installed"))
			}
		} else {
			if err := store.GetStore().SetTaskStatus("airgap-install", finalError.Error(), "failed"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set error on install task status"))
			}
			if err := store.GetStore().SetAppInstallState(pendingApp.ID, "airgap_upload_error"); err != nil {
				logger.Error(errors.Wrap(err, "failed to set app status to error"))
			}
		}
	}()

	if err := store.GetStore().SetAppIsAirgap(pendingApp.ID, true); err != nil {
		return errors.Wrap(err, "failed to set app is airgap")
	}

	// Extract it
	if err := store.GetStore().SetTaskStatus("airgap-install", "Extracting files...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	// we seem to need a lot of temp dirs here... maybe too many?
	archiveDir, err := version.ExtractArchiveToTempDirectory(airgapBundle)
	if err != nil {
		return errors.Wrap(err, "failed to extract archive")
	}
	defer os.RemoveAll(archiveDir)

	// extract the release
	workspace, err := ioutil.TempDir("", "kots-airgap")
	if err != nil {
		return errors.Wrap(err, "failed to create workspace")
	}
	defer os.RemoveAll(workspace)

	releaseDir, err := extractAppRelease(workspace, archiveDir)
	if err != nil {
		return errors.Wrap(err, "failed to extract app dir")
	}

	tmpRoot, err := ioutil.TempDir("", "kots")
	if err != nil {
		return errors.Wrap(err, "failed to create temp root")
	}
	defer os.RemoveAll(tmpRoot)

	if err := store.GetStore().SetTaskStatus("airgap-install", "Reading license data...", "running"); err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(pendingApp.LicenseData), nil, nil)
	if err != nil {
		return errors.Wrap(err, "failed to read pending license data")
	}
	license := obj.(*kotsv1beta1.License)

	licenseFile, err := ioutil.TempFile("", "kotadm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp file")
	}
	if err := ioutil.WriteFile(licenseFile.Name(), []byte(pendingApp.LicenseData), 0644); err != nil {
		os.Remove(licenseFile.Name())
		return errors.Wrapf(err, "failed to write license to temp file")
	}

	pipeReader, pipeWriter := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			if err := store.GetStore().SetTaskStatus("airgap-install", scanner.Text(), "running"); err != nil {
				logger.Error(err)
			}
		}
		pipeReader.CloseWithError(scanner.Err())
	}()

	appNamespace := os.Getenv("POD_NAMESPACE")
	if os.Getenv("KOTSADM_TARGET_NAMESPACE") != "" {
		appNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")
	}

	pullOptions := pull.PullOptions{
		Downstreams:         []string{"this-cluster"},
		LocalPath:           releaseDir,
		Namespace:           appNamespace,
		LicenseFile:         licenseFile.Name(),
		AirgapRoot:          archiveDir,
		Silent:              true,
		ExcludeKotsKinds:    true,
		RootDir:             tmpRoot,
		ExcludeAdminConsole: true,
		RewriteImages:       true,
		ReportWriter:        pipeWriter,
		RewriteImageOptions: pull.RewriteImageOptions{
			ImageFiles: filepath.Join(archiveDir, "images"),
			Host:       registryHost,
			Namespace:  namespace,
			Username:   username,
			Password:   password,
		},
		AppSlug:     pendingApp.Slug,
		AppSequence: 0,
	}

	if _, err := pull.Pull(fmt.Sprintf("replicated://%s", license.Spec.AppSlug), pullOptions); err != nil {
		return errors.Wrap(err, "failed to pull")
	}

	if err := store.GetStore().AddAppToAllDownstreams(pendingApp.ID); err != nil {
		return errors.Wrap(err, "failed to add app to all downstreams")
	}

	a, err := store.GetStore().GetApp(pendingApp.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get app from pending app")
	}

	if password == registrytypes.PasswordMask {
		// On initial install, registry info can be copied from kotsadm config,
		// and password in this case will not be included in the request.
		kotsadmSettings, err := registry.GetKotsadmRegistry()
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to load kotsadm config"))
		} else if kotsadmSettings.Hostname == registryHost {
			password = kotsadmSettings.Password
		}
	}

	if err := store.GetStore().UpdateRegistry(pendingApp.ID, registryHost, username, password, namespace); err != nil {
		return errors.Wrap(err, "failed to update registry")
	}

	// yes, again in case of errors
	if err := store.GetStore().SetAppIsAirgap(pendingApp.ID, true); err != nil {
		return errors.Wrap(err, "failed to set app is airgap the second time")
	}

	newSequence, err := version.CreateFirstVersion(a.ID, tmpRoot, "Airgap Upload")
	if err != nil {
		return errors.Wrap(err, "failed to create new version")
	}

	if err := preflight.Run(pendingApp.ID, newSequence, tmpRoot); err != nil {
		return errors.Wrap(err, "failed to start preflights")
	}

	return nil
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
		err := extractTGZArchive(filepath.Join(airgapDir, file.Name()), destDir)
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

// todo, figure out why this doesn't use the mholt tgz archiver that we
// use elsewhere in kots
func extractTGZArchive(tgzFile string, destDir string) error {
	fileReader, err := os.Open(tgzFile)
	if err != nil {
		return errors.Wrap(err, "failed to open tgz file")
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
			return errors.Wrap(err, "failed to read tar data")
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
