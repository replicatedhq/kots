package download

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
)

type DownloadOptions struct {
	Namespace  string
	Kubeconfig string
	Overwrite  bool
	Silent     bool
}

func Download(appSlug string, path string, downloadOptions DownloadOptions) error {
	log := logger.NewLogger()
	if downloadOptions.Silent {
		log.Silence()
	}

	log.ActionWithSpinner("Connecting to cluster")

	podName, err := k8sutil.FindKotsadm(downloadOptions.Namespace)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to find kotsadm pod")
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	_, errChan, err := k8sutil.PortForward(downloadOptions.Kubeconfig, 3000, 3000, downloadOptions.Namespace, podName, false, stopCh, log)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to start port forwarding")
	}

	go func() {
		select {
		case err := <-errChan:
			if err != nil {
				log.Error(err)
			}
		case <-stopCh:
		}
	}()

	authSlug, err := auth.GetOrCreateAuthSlug(downloadOptions.Namespace)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to get kotsadm auth slug")
	}

	url := fmt.Sprintf("http://localhost:3000/api/v1/kots/%s", appSlug)
	newRequest, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to create download request")
	}
	newRequest.Header.Add("Authorization", authSlug)

	resp, err := http.DefaultClient.Do(newRequest)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to get from kotsadm")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.FinishSpinnerWithError()
		return errors.Errorf("unexpected status code from %s: %s", url, resp.Status)
	}

	tmpFile, err := ioutil.TempFile("", "kots")
	if err != nil {
		log.FinishSpinner()
		return errors.Wrap(err, "failed to create temp file")
	}
	defer os.Remove(tmpFile.Name())

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		log.FinishSpinner()
		return errors.Wrap(err, "failed to write archive")
	}
	tmpFile.Close()

	// Delete the destination, if needed and requested
	if _, err := os.Stat(path); err == nil {
		if downloadOptions.Overwrite {
			if err := os.RemoveAll(path); err != nil {
				return errors.Wrap(err, "failed to delete existing download")
			}
		} else {
			log.FinishSpinner()
			log.ActionWithoutSpinner("")
			log.Info("Directory %s already exists. You can re-run this command with --overwrite to automatically overwrite it", path)
			log.ActionWithoutSpinner("")
			return errors.Errorf("directory already exists at %s", path)
		}
	}

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Unarchive(tmpFile.Name(), path); err != nil {
		return errors.Wrap(err, "failed to extract tar gz")
	}

	log.FinishSpinner()

	return nil
}
