package snapshot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type InstanceBackupOptions struct {
	Namespace             string
	KubernetesConfigFlags *genericclioptions.ConfigFlags
	Silent                bool
}

func InstanceBackup(instanceBackupOptions InstanceBackupOptions) error {
	log := logger.NewLogger()
	if instanceBackupOptions.Silent {
		log.Silence()
	}

	log.ActionWithSpinner("Connecting to cluster")

	clientset, err := k8sutil.GetClientset(instanceBackupOptions.KubernetesConfigFlags)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to get clientset")
	}

	podName, err := k8sutil.FindKotsadm(clientset, instanceBackupOptions.Namespace)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to find kotsadm pod")
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	localPort, errChan, err := k8sutil.PortForward(instanceBackupOptions.KubernetesConfigFlags, 0, 3000, instanceBackupOptions.Namespace, podName, false, stopCh, log)
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

	authSlug, err := auth.GetOrCreateAuthSlug(instanceBackupOptions.KubernetesConfigFlags, instanceBackupOptions.Namespace)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to get kotsadm auth slug")
	}

	url := fmt.Sprintf("http://localhost:%d/api/v1/snapshot/backup", localPort)

	newRequest, err := http.NewRequest("POST", url, nil)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to create instance snapshot backup request")
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

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to read server response")
	}

	type BackupResponse struct {
		Success    bool   `json:"success"`
		BackupName string `json:"backupName,omitempty"`
		Error      string `json:"error,omitempty"`
	}
	var backupResponse BackupResponse
	if err := json.Unmarshal(respBody, &backupResponse); err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to unmarshal response")
	}

	if backupResponse.Error != "" {
		log.FinishSpinnerWithError()
		return errors.New(backupResponse.Error)
	}

	log.FinishSpinner()
	log.ActionWithoutSpinner(fmt.Sprintf("Backup request has been created. Backup name is %s", backupResponse.BackupName))

	return nil
}
