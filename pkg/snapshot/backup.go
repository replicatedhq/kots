package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	snapshottypes "github.com/replicatedhq/kots/pkg/kotsadmsnapshot/types"
	"github.com/replicatedhq/kots/pkg/logger"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type CreateInstanceBackupOptions struct {
	Namespace string
	Wait      bool
	Silent    bool
}

type ListInstanceBackupsOptions struct {
	Namespace string
}

type VeleroRBACResponse struct {
	Success                     bool   `json:"success"`
	Error                       string `json:"error,omitempty"`
	KotsadmNamespace            string `json:"kotsadmNamespace,omitempty"`
	KotsadmRequiresVeleroAccess bool   `json:"kotsadmRequiresVeleroAccess,omitempty"`
}

type BackupResponse struct {
	Success    bool   `json:"success"`
	BackupName string `json:"backupName,omitempty"`
	Error      string `json:"error,omitempty"`
}

func CreateInstanceBackup(ctx context.Context, options CreateInstanceBackupOptions) (*BackupResponse, error) {
	log := logger.NewCLILogger(os.Stdout)
	if options.Silent {
		log.Silence()
	}

	log.ActionWithSpinner("Connecting to cluster")

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	getPodName := func() (string, error) {
		return k8sutil.FindKotsadm(clientset, options.Namespace)
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	localPort, errChan, err := k8sutil.PortForward(0, 3000, options.Namespace, getPodName, false, stopCh, log)
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to start port forwarding")
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

	authSlug, err := auth.GetOrCreateAuthSlug(clientset, options.Namespace)
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to get kotsadm auth slug")
	}

	url := fmt.Sprintf("http://localhost:%d/api/v1/snapshot/backup", localPort)

	newRequest, err := http.NewRequest("POST", url, nil)
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to create instance snapshot backup request")
	}
	newRequest.Header.Add("Authorization", authSlug)

	resp, err := http.DefaultClient.Do(newRequest)
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to get from kotsadm")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to read server response")
	}

	if resp.StatusCode != http.StatusOK {
		log.FinishSpinnerWithError()
		if resp.StatusCode == http.StatusConflict {
			veleroRBACResponse := VeleroRBACResponse{}
			if err := json.Unmarshal(respBody, &veleroRBACResponse); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal velero rbac response")
			}
			if veleroRBACResponse.KotsadmRequiresVeleroAccess {
				log.ActionWithoutSpinner("Velero Namespace Access Required")
				log.ActionWithoutSpinner("We’ve detected that the Admin Console is running with minimal role-based-access-control (RBAC) privileges, meaning that the Admin Console is limited to a single namespace. To use the snapshots functionality, the Admin Console requires access to the namespace Velero is installed in. Please make sure Velero is installed, then use the following command to provide the Admin Console with the necessary permissions to access it:\n")
				log.Info("kubectl kots velero ensure-permissions --namespace %s --velero-namespace <velero-namespace>", veleroRBACResponse.KotsadmNamespace)
				log.Info("* Note: Please replace `<velero-namespace>` with the actual namespace Velero is installed in, which is 'velero' by default.\n")
				return &BackupResponse{
					Error: "unable to access velero due to minimal RBAC privileges",
				}, nil
			}
		}
		return nil, errors.Errorf("unexpected status code from %s: %s", url, resp.Status)
	}

	var backupResponse BackupResponse
	if err := json.Unmarshal(respBody, &backupResponse); err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to unmarshal response")
	}

	if backupResponse.Error != "" {
		log.FinishSpinnerWithError()
		return nil, errors.New(backupResponse.Error)
	}

	if options.Wait {
		// wait for backup to complete
		backup, err := waitForVeleroBackupCompleted(ctx, clientset, backupResponse.BackupName, options.Namespace)
		if err != nil {
			if backup != nil {
				errMsg := fmt.Sprintf("backup failed with %d errors and %d warnings.", backup.Status.Errors, backup.Status.Warnings)
				log.FinishSpinnerWithError()
				log.ActionWithoutSpinner("%s", errMsg)
				return nil, errors.Wrap(err, errMsg)
			}
			log.FinishSpinnerWithError()
			return nil, errors.Wrap(err, "failed to wait for velero backup completed")
		}

		log.FinishSpinner()
		log.ActionWithoutSpinner("Backup completed successfully. Backup name is %s", backupResponse.BackupName)
	} else {
		log.FinishSpinner()
		log.ActionWithoutSpinner("Backup is in progress. Backup name is %s", backupResponse.BackupName)
	}

	return &backupResponse, nil
}

func ListInstanceBackups(ctx context.Context, options ListInstanceBackupsOptions) ([]velerov1.Backup, error) {
	b, err := ListAllBackups(ctx, options)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get backup list")
	}

	backups := []velerov1.Backup{}

	for _, backup := range b {
		if !snapshottypes.IsInstanceBackup(backup) {
			continue
		}

		if options.Namespace != "" && backup.Annotations["kots.io/kotsadm-deploy-namespace"] != options.Namespace {
			continue
		}

		backups = append(backups, backup)
	}

	return backups, nil
}

func ListAllBackups(ctx context.Context, options ListInstanceBackupsOptions) ([]velerov1.Backup, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s client")
	}
	veleroNamespace, err := DetectVeleroNamespace(ctx, clientset, options.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to detect velero namespace")
	}
	if veleroNamespace == "" {
		return nil, errors.New("velero not found")
	}

	veleroClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	var backupList velerov1.BackupList
	err = veleroClient.List(ctx, &backupList, kbclient.InNamespace(veleroNamespace))
	if err != nil {
		return nil, errors.Wrap(err, "failed to list backups")
	}

	backups := []velerov1.Backup{}
	backups = append(backups, backupList.Items...)

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

	veleroClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client")
	}

	for {
		var backup velerov1.Backup
		err := veleroClient.Get(ctx, k8stypes.NamespacedName{Namespace: veleroNamespace, Name: backupName}, &backup)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get backup")
		}

		switch backup.Status.Phase {
		case velerov1.BackupPhaseCompleted:
			return &backup, nil
		case velerov1.BackupPhaseFailed:
			return &backup, errors.New("backup failed")
		case velerov1.BackupPhasePartiallyFailed:
			return &backup, errors.New("backup partially failed")
		default:
			// in progress
		}

		time.Sleep(time.Second)
	}
}
