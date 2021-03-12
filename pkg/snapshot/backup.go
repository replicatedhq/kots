package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type CreateInstanceBackupOptions struct {
	Namespace             string
	KubernetesConfigFlags *genericclioptions.ConfigFlags
	Wait                  bool
}

type ListInstanceBackupsOptions struct {
	Namespace             string
	KubernetesConfigFlags *genericclioptions.ConfigFlags
}

type VeleroRBACResponse struct {
	Success                     bool   `json:"success"`
	Error                       string `json:"error,omitempty"`
	KotsadmNamespace            string `json:"kotsadmNamespace,omitempty"`
	KotsadmRequiresVeleroAccess bool   `json:"kotsadmRequiresVeleroAccess,omitempty"`
}

func CreateInstanceBackup(ctx context.Context, options CreateInstanceBackupOptions) error {
	log := logger.NewCLILogger()
	log.ActionWithSpinner("Connecting to cluster")

	clientset, err := k8sutil.GetClientset(options.KubernetesConfigFlags)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to get clientset")
	}

	podName, err := k8sutil.FindKotsadm(clientset, options.Namespace)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to find kotsadm pod")
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	localPort, errChan, err := k8sutil.PortForward(options.KubernetesConfigFlags, 0, 3000, options.Namespace, podName, false, stopCh, log)
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

	log.FinishSpinner()
	log.ActionWithSpinner("Creating Backup")

	authSlug, err := auth.GetOrCreateAuthSlug(options.KubernetesConfigFlags, options.Namespace)
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

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to read server response")
	}

	if resp.StatusCode != http.StatusOK {
		log.FinishSpinnerWithError()
		if resp.StatusCode == http.StatusConflict {
			veleroRBACResponse := VeleroRBACResponse{}
			if err := json.Unmarshal(respBody, &veleroRBACResponse); err != nil {
				return errors.Wrap(err, "failed to unmarshal velero rbac response")
			}
			if veleroRBACResponse.KotsadmRequiresVeleroAccess {
				log.ActionWithoutSpinner("Velero Namespace Access Required")
				log.ActionWithoutSpinner("We’ve detected that the Admin Console is running with minimal role-based-access-control (RBAC) privileges, meaning that the Admin Console is limited to a single namespace. To use the snapshots functionality, the Admin Console requires access to the namespace Velero is installed in. Please make sure Velero is installed, then use the following command to provide the Admin Console with the necessary permissions to access it:\n")
				log.Info("kubectl kots velero ensure-permissions --namespace %s --velero-namespace <velero-namespace>", veleroRBACResponse.KotsadmNamespace)
				log.Info("* Note: Please replace `<velero-namespace>` with the actual namespace Velero is installed in, which is 'velero' by default.\n")
				return nil
			}
		}
		return errors.Errorf("unexpected status code from %s: %s", url, resp.Status)
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

	if options.Wait {
		// wait for backup to complete
		backup, err := waitForVeleroBackupCompleted(ctx, clientset, backupResponse.BackupName, options.Namespace)
		if err != nil {
			if backup != nil {
				errMsg := fmt.Sprintf("backup failed with %d errors and %d warnings.", backup.Status.Errors, backup.Status.Warnings)
				log.FinishSpinnerWithError()
				log.ActionWithoutSpinner(errMsg)
				return errors.Wrap(err, errMsg)
			}
			log.FinishSpinnerWithError()
			return errors.Wrap(err, "failed to wait for velero backup completed")
		}

		log.FinishSpinner()
		log.ActionWithoutSpinner(fmt.Sprintf("Backup completed successfully. Backup name is %s", backupResponse.BackupName))
	} else {
		log.FinishSpinner()
		log.ActionWithoutSpinner(fmt.Sprintf("Backup is in progress. Backup name is %s", backupResponse.BackupName))
	}

	return nil
}

func ListInstanceBackups(ctx context.Context, options ListInstanceBackupsOptions) ([]velerov1.Backup, error) {
	clientset, err := k8sutil.GetClientset(options.KubernetesConfigFlags)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}
	veleroNamespace, err := DetectVeleroNamespace(ctx, clientset, options.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to detect velero namespace")
	}
	if veleroNamespace == "" {
		return nil, errors.New("velero not found")
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	b, err := veleroClient.Backups(veleroNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list backups")
	}

	backups := []velerov1.Backup{}

	for _, backup := range b.Items {
		if backup.Annotations["kots.io/instance"] != "true" {
			continue
		}

		if options.Namespace != "" && backup.Annotations["kots.io/kotsadm-deploy-namespace"] != options.Namespace {
			continue
		}

		backups = append(backups, backup)
	}

	return backups, nil
}

func waitForVeleroBackupCompleted(ctx context.Context, clientset kubernetes.Interface, backupName string, namespace string) (*velerov1.Backup, error) {
	veleroNamespace, err := DetectVeleroNamespace(ctx, clientset, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to detect velero namespace")
	}
	if veleroNamespace == "" {
		return nil, errors.New("velero not found")
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	for {
		backup, err := veleroClient.Backups(veleroNamespace).Get(ctx, backupName, metav1.GetOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get backup")
		}

		switch backup.Status.Phase {
		case velerov1.BackupPhaseCompleted:
			return backup, nil
		case velerov1.BackupPhaseFailed:
			return backup, errors.New("backup failed")
		case velerov1.BackupPhasePartiallyFailed:
			return backup, errors.New("backup partially failed")
		default:
			// in progress
		}

		time.Sleep(time.Second)
	}
}
