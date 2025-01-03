package snapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	snapshottypes "github.com/replicatedhq/kots/pkg/kotsadmsnapshot/types"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type RestoreInstanceBackupOptions struct {
	BackupName          string
	ExcludeAdminConsole bool
	ExcludeApps         bool
	WaitForApps         bool
	VeleroNamespace     string
	Silent              bool
}

type ListInstanceRestoresOptions struct {
	Namespace string
}

func RestoreInstanceBackup(ctx context.Context, options RestoreInstanceBackupOptions) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s clientset")
	}

	veleroNamespace := options.VeleroNamespace
	if veleroNamespace == "" {
		var err error
		veleroNamespace, err = DetectVeleroNamespace(ctx, clientset, "")
		if err != nil {
			return errors.Wrap(err, "failed to detect velero namespace")
		}
		if veleroNamespace == "" {
			return errors.New("velero not found")
		}
	}

	// get the backup
	veleroClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to create velero client")
	}

	var backup velerov1.Backup
	err = veleroClient.Get(ctx, k8stypes.NamespacedName{Namespace: veleroNamespace, Name: options.BackupName}, &backup)
	if err != nil {
		return errors.Wrap(err, "failed to find backup")
	}

	// make sure this is an instance backup
	if !snapshottypes.IsInstanceBackup(backup) {
		return errors.Wrap(err, "backup provided is not an instance backup")
	}

	if snapshottypes.GetInstanceBackupType(backup) != snapshottypes.InstanceBackupTypeLegacy {
		return errors.New("only legacy type instance backups are restorable")
	}

	kotsadmImage, ok := backup.Annotations["kots.io/kotsadm-image"]
	if !ok {
		return errors.Wrap(err, "failed to find kotsadm image annotation")
	}

	kotsadmNamespace, _ := backup.Annotations["kots.io/kotsadm-deploy-namespace"]
	if kotsadmNamespace == "" {
		return errors.Wrap(err, "failed to find kotsadm deploy namespace annotation")
	}

	// make sure backup is restorable/complete
	switch backup.Status.Phase {
	case velerov1.BackupPhaseCompleted:
		break
	case velerov1.BackupPhaseFailed, velerov1.BackupPhasePartiallyFailed:
		return errors.Wrap(err, "cannot restore a failed backup")
	default:
		return errors.Wrap(err, "backup is still in progress")
	}

	log := logger.NewCLILogger(os.Stdout)
	if options.Silent {
		log.Silence()
	}

	if !options.ExcludeAdminConsole {
		log.ActionWithSpinner("Deleting Admin Console")

		isKurl, err := kurl.IsKurl(clientset)
		if err != nil {
			return errors.Wrap(err, "failed to check if cluster is kurl")
		}

		// delete all kotsadm objects before creating the restore
		err = k8sutil.DeleteKotsadm(ctx, clientset, kotsadmNamespace, isKurl)
		if err != nil {
			log.FinishSpinnerWithError()
			return errors.Wrap(err, "failed to delete kotsadm objects")
		}

		log.FinishSpinner()
		log.ActionWithSpinner("Restoring Admin Console")

		// create a restore for kotsadm objects
		restore := &velerov1.Restore{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: veleroNamespace,
				Name:      fmt.Sprintf("%s.kotsadm", backup.Name),
				Annotations: map[string]string{
					snapshottypes.InstanceBackupAnnotation: "true",
					"kots.io/kotsadm-image":                kotsadmImage,
					"kots.io/kotsadm-deploy-namespace":     kotsadmNamespace,
				},
			},
			Spec: velerov1.RestoreSpec{
				BackupName: backup.Name,
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						kotsadmtypes.KotsadmKey: kotsadmtypes.KotsadmLabelValue, // restoring applications is in a separate step after kotsadm spins up
					},
				},
				RestorePVs:              pointer.Bool(true),
				IncludeClusterResources: pointer.Bool(true),
			},
		}

		// delete existing restore object (if exists)
		err = veleroClient.Delete(ctx, restore)
		if err != nil && !strings.Contains(err.Error(), "not found") {
			log.FinishSpinnerWithError()
			return errors.Wrapf(err, "failed to delete restore %s", restore.ObjectMeta.Name)
		}

		// create new restore object
		err = veleroClient.Create(ctx, restore)
		if err != nil {
			log.FinishSpinnerWithError()
			return errors.Wrap(err, "failed to create restore")
		}

		// wait for restore to complete
		restore, err = waitForVeleroRestoreCompleted(ctx, veleroNamespace, restore.ObjectMeta.Name)
		if err != nil {
			if restore != nil {
				errMsg := fmt.Sprintf("Admin Console restore failed with %d errors and %d warnings.", restore.Status.Errors, restore.Status.Warnings)
				log.FinishSpinnerWithError()
				log.ActionWithoutSpinner(errMsg)
				return errors.Wrap(err, errMsg)
			}
			log.FinishSpinnerWithError()
			return errors.Wrap(err, "failed to wait for velero restore completed")
		}

		// wait for kotsadm to start up
		timeout, err := time.ParseDuration("10m")
		if err != nil {
			log.FinishSpinnerWithError()
			return errors.Wrap(err, "failed to parse timeout value")
		}
		_, err = k8sutil.WaitForKotsadm(clientset, kotsadmNamespace, timeout)
		if err != nil {
			log.FinishSpinnerWithError()
			return errors.Wrap(err, "failed to wait for kotsadm")
		}

		log.FinishSpinner()
	}

	if !options.ExcludeApps {
		log.ActionWithSpinner("Restoring Applications")

		// make sure kotsadm is up and running
		timeout, err := time.ParseDuration("10m")
		if err != nil {
			log.FinishSpinnerWithError()
			return errors.Wrap(err, "failed to parse timeout value")
		}
		kotsadmPodName, err := k8sutil.WaitForKotsadm(clientset, kotsadmNamespace, timeout)
		if err != nil {
			log.FinishSpinnerWithError()
			return errors.Wrap(err, "failed to wait for kotsadm")
		}

		// initiate kotsadm applications restore
		err = initiateKotsadmApplicationsRestore(backup.Name, kotsadmNamespace, kotsadmPodName, log)
		if err != nil {
			log.FinishSpinnerWithError()
			return errors.Wrap(err, "failed to restore kotsadm applications")
		}

		if options.WaitForApps {
			// wait for applications restore to finish
			err = waitForKotsadmApplicationsRestore(backup.Name, kotsadmNamespace, kotsadmPodName, log)
			if err != nil {
				if _, ok := errors.Cause(err).(*kotsadmtypes.ErrorAppsRestore); ok {
					log.FinishSpinnerWithError()
					return errors.Errorf("failed to restore kotsadm applications: %s", err)
				}
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to wait for kotsadm applications restore")
			}
		}

		log.FinishSpinner()
	}

	// both admin console and apps were restored
	if !options.ExcludeAdminConsole && !options.ExcludeApps {
		if options.WaitForApps {
			log.ActionWithoutSpinner("Restore completed successfully.")
		} else {
			log.ActionWithoutSpinner("Admin Console restored successfully. Applications restore is still in progress.")
		}
		return nil
	}

	// only the admin console was restored
	if !options.ExcludeAdminConsole && options.ExcludeApps {
		log.ActionWithoutSpinner("Admin Console restored successfully.")
		return nil
	}

	// only the applications were restored
	if options.ExcludeAdminConsole && !options.ExcludeApps {
		if options.WaitForApps {
			log.ActionWithoutSpinner("Applications restored successfully.")
		} else {
			log.ActionWithoutSpinner("Applications restore initiated successfully but is still in progress.")
		}
		return nil
	}

	return nil
}

func ListInstanceRestores(ctx context.Context, options ListInstanceRestoresOptions) ([]velerov1.Restore, error) {
	clientset, err := k8sutil.GetClientset()
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

	veleroClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client")
	}

	var restoreList velerov1.RestoreList
	err = veleroClient.List(ctx, &restoreList, kbclient.InNamespace(veleroNamespace))
	if err != nil {
		return nil, errors.Wrap(err, "failed to list restores")
	}

	restores := []velerov1.Restore{}

	for _, restore := range restoreList.Items {
		if restore.Annotations[snapshottypes.InstanceBackupAnnotation] != "true" {
			continue
		}

		if options.Namespace != "" && restore.Annotations["kots.io/kotsadm-deploy-namespace"] != options.Namespace {
			continue
		}

		restores = append(restores, restore)
	}

	return restores, nil
}

func waitForVeleroRestoreCompleted(ctx context.Context, veleroNamespace string, restoreName string) (*velerov1.Restore, error) {
	veleroClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client")
	}

	for {
		var restore velerov1.Restore
		err := veleroClient.Get(ctx, k8stypes.NamespacedName{Namespace: veleroNamespace, Name: restoreName}, &restore)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get restore")
		}

		switch restore.Status.Phase {
		case velerov1.RestorePhaseCompleted:
			return &restore, nil
		case velerov1.RestorePhaseFailed:
			return &restore, errors.New("restore failed")
		case velerov1.RestorePhasePartiallyFailed:
			return &restore, errors.New("restore partially failed")
		default:
			// in progress
		}

		time.Sleep(time.Second)
	}
}

func initiateKotsadmApplicationsRestore(backupID string, kotsadmNamespace string, kotsadmPodName string, log *logger.CLILogger) error {
	getPodName := func() (string, error) {
		return kotsadmPodName, nil
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	localPort, errChan, err := k8sutil.PortForward(0, 3000, kotsadmNamespace, getPodName, false, stopCh, log)
	if err != nil {
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

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s clientset")
	}

	authSlug, err := auth.GetOrCreateAuthSlug(clientset, kotsadmNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to get kotsadm auth slug")
	}

	url := fmt.Sprintf("http://localhost:%d/api/v1/snapshot/%s/restore-apps", localPort, backupID)

	requestPayload := map[string]interface{}{
		"restoreAll": true,
	}
	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return errors.Wrap(err, "failed to marshal request json")
	}

	newRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}
	newRequest.Header.Add("Authorization", authSlug)

	resp, err := http.DefaultClient.Do(newRequest)
	if err != nil {
		return errors.Wrap(err, "failed to get from kotsadm")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected status code from %s: %s", url, resp.Status)
	}

	return nil
}

func waitForKotsadmApplicationsRestore(backupID string, kotsadmNamespace string, kotsadmPodName string, log *logger.CLILogger) error {
	getPodName := func() (string, error) {
		return kotsadmPodName, nil
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	localPort, errChan, err := k8sutil.PortForward(0, 3000, kotsadmNamespace, getPodName, false, stopCh, log)
	if err != nil {
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

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s clientset")
	}

	authSlug, err := auth.GetOrCreateAuthSlug(clientset, kotsadmNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to get kotsadm auth slug")
	}

	url := fmt.Sprintf("http://localhost:%d/api/v1/snapshot/%s/apps-restore-status", localPort, backupID)

	for {
		requestPayload := map[string]interface{}{
			"checkAll": true,
		}
		requestBody, err := json.Marshal(requestPayload)
		if err != nil {
			return errors.Wrap(err, "failed to marshal request json")
		}
		newRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
		if err != nil {
			return errors.Wrap(err, "failed to create request")
		}
		newRequest.Header.Add("Authorization", authSlug)

		resp, err := http.DefaultClient.Do(newRequest)
		if err != nil {
			return errors.Wrap(err, "failed to get from kotsadm")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return errors.Errorf("unexpected status code from %s: %s", url, resp.Status)
		}

		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "failed to read server response")
		}

		type AppRestoreStatus struct {
			AppSlug       string                      `json:"appSlug"`
			RestoreDetail snapshottypes.RestoreDetail `json:"restoreDetail"`
		}
		type AppsRestoreStatusResponse struct {
			Statuses []AppRestoreStatus `json:"statuses"`
			Error    string             `json:"error,omitempty"`
		}
		var appsRestoreStatusResponse AppsRestoreStatusResponse
		if err := json.Unmarshal(respBody, &appsRestoreStatusResponse); err != nil {
			return errors.Wrap(err, "failed to unmarshal response")
		}

		if appsRestoreStatusResponse.Error != "" {
			return errors.New(appsRestoreStatusResponse.Error)
		}

		inProgress := false
		errs := []string{}

		for _, s := range appsRestoreStatusResponse.Statuses {
			switch s.RestoreDetail.Phase {
			case velerov1.RestorePhaseCompleted:
				break
			case velerov1.RestorePhaseFailed, velerov1.RestorePhasePartiallyFailed:
				errMsg := fmt.Sprintf("restore failed for app %s with %d errors and %d warnings", s.AppSlug, len(s.RestoreDetail.Errors), len(s.RestoreDetail.Warnings))
				errs = append(errs, errMsg)
				break
			default:
				inProgress = true
			}
		}

		if !inProgress {
			if len(errs) == 0 {
				return nil
			} else {
				errMsg := strings.Join(errs, " AND ")
				return &kotsadmtypes.ErrorAppsRestore{
					Message: errMsg,
				}
			}
		}

		time.Sleep(time.Second * 2)
	}
}
