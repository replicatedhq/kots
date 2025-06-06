package download

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
)

type DownloadOptions struct {
	Namespace             string
	Overwrite             bool
	Silent                bool
	DecryptPasswordValues bool
	Current               bool
	Sequence              int64
}

func Download(appSlug string, path string, downloadOptions DownloadOptions) error {
	log := logger.NewCLILogger(os.Stdout)
	if downloadOptions.Silent {
		log.Silence()
	}

	log.ActionWithSpinner("Connecting to cluster")

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to get clientset")
	}

	getPodName := func() (string, error) {
		return k8sutil.FindKotsadm(clientset, downloadOptions.Namespace)
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	localPort, errChan, err := k8sutil.PortForward(0, 3000, downloadOptions.Namespace, getPodName, false, stopCh, log)
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

	authSlug, err := auth.GetOrCreateAuthSlug(clientset, downloadOptions.Namespace)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to get kotsadm auth slug")
	}

	url := fmt.Sprintf("http://localhost:%d/api/v1/download?slug=%s", localPort, appSlug)
	if downloadOptions.DecryptPasswordValues {
		url = fmt.Sprintf("%s&decryptPasswordValues=1", url)
	}
	if downloadOptions.Current {
		url = fmt.Sprintf("%s&current=1", url)
	}
	if downloadOptions.Sequence != -1 {
		url = fmt.Sprintf("%s&sequence=%d", url, downloadOptions.Sequence)
	}

	newRequest, err := util.NewRequest("GET", url, nil)
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
		if resp.StatusCode == http.StatusNotFound {
			return errors.Errorf("app with slug %s not found", appSlug)
		} else {
			return errors.Errorf("unexpected status code from %s: %s", url, resp.Status)
		}
	}

	tmpFile, err := os.CreateTemp("", "kots")
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
			log.Error(errors.Errorf("Directory %s already exists. You can re-run this command with --overwrite to automatically overwrite it", path))
			log.ActionWithoutSpinner("")
			return errors.Errorf("directory already exists at %s", path)
		}
	}

	if err := util.ExtractTGZArchive(tmpFile.Name(), path); err != nil {
		return errors.Wrap(err, "failed to extract tar gz")
	}

	log.FinishSpinner()

	return nil
}
