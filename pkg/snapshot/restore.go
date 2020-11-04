package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type CreateInstanceRestoreOptions struct {
	BackupName            string
	KubernetesConfigFlags *genericclioptions.ConfigFlags
}

type ListInstanceRestoresOptions struct {
	Namespace string
}

func CreateInstanceRestore(options CreateInstanceRestoreOptions) (*velerov1.Restore, error) {
	bsl, err := findBackupStoreLocation()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get velero namespace")
	}

	veleroNamespace := bsl.Namespace

	// get the backup
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create velero clientset")
	}

	backup, err := veleroClient.Backups(veleroNamespace).Get(context.TODO(), options.BackupName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to find backup")
	}

	// make sure this is an instance backup
	if backup.Annotations["kots.io/instance"] != "true" {
		return nil, errors.Wrap(err, "backup provided is not an instance backup")
	}

	kotsadmImage, ok := backup.Annotations["kots.io/kotsadm-image"]
	if !ok {
		return nil, errors.Wrap(err, "failed to find kotsadm image annotation")
	}

	kotsadmNamespace, ok := backup.Annotations["kots.io/kotsadm-deploy-namespace"]
	if !ok {
		return nil, errors.Wrap(err, "failed to find kotsadm deploy namespace annotation")
	}

	// make sure backup is restorable/complete
	switch backup.Status.Phase {
	case velerov1.BackupPhaseCompleted:
		break
	case velerov1.BackupPhaseFailed, velerov1.BackupPhasePartiallyFailed:
		return nil, errors.Wrap(err, "cannot restore a failed backup")
	default:
		return nil, errors.Wrap(err, "backup is still in progress")
	}

	log := logger.NewLogger()
	log.ActionWithSpinner("Deleting Admin Console")

	// delete all kotsadm objects before creating the restore
	clientset, err := k8sutil.GetClientset(options.KubernetesConfigFlags)
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}
	err = k8sutil.DeleteKotsadm(clientset, kotsadmNamespace)
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to delete kotsadm objects")
	}

	log.FinishSpinner()
	log.ActionWithSpinner("Restoring Admin Console")

	// create a restore for kotsadm objects
	trueVal := true
	restore := &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: veleroNamespace,
			Name:      fmt.Sprintf("%s.kotsadm", options.BackupName),
			Annotations: map[string]string{
				"kots.io/instance":                 "true",
				"kots.io/kotsadm-image":            kotsadmImage,
				"kots.io/kotsadm-deploy-namespace": kotsadmNamespace,
			},
		},
		Spec: velerov1.RestoreSpec{
			BackupName: options.BackupName,
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					kotsadmtypes.KotsadmKey: kotsadmtypes.KotsadmLabelValue, // application restores are in a separate step
				},
			},
			RestorePVs:              &trueVal,
			IncludeClusterResources: &trueVal,
		},
	}

	// delete existing restore object (if exists)
	err = veleroClient.Restores(veleroNamespace).Delete(context.TODO(), restore.ObjectMeta.Name, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		log.FinishSpinnerWithError()
		return nil, errors.Wrapf(err, "failed to delete restore %s", restore.ObjectMeta.Name)
	}

	// create new restore object
	_, err = veleroClient.Restores(veleroNamespace).Create(context.TODO(), restore, metav1.CreateOptions{})
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to create restore")
	}

	// wait for kotsadm to start
	timeout, err := time.ParseDuration("10m")
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to parse timeout value")
	}
	kotsadmPodName, err := k8sutil.WaitForKotsadm(clientset, kotsadmNamespace, timeout)
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to wait for kotsadm")
	}

	log.FinishSpinner()
	log.ActionWithSpinner("Restoring Applications")

	// initiate kotsadm applications restore
	err = initiateKotsadmApplicationsRestore(options.BackupName, kotsadmNamespace, kotsadmPodName, options.KubernetesConfigFlags, log)
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to restore kotsadm applications")
	}

	// wait for applications restore to finish
	err = waitForKotsadmApplicationsRestore(kotsadmNamespace, kotsadmPodName, options.KubernetesConfigFlags, log)
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to wait for kotsadm applications restore")
	}

	log.FinishSpinner()
	log.ActionWithoutSpinner("Restore completed successfully.")

	return restore, nil
}

func ListInstanceRestores(options ListInstanceRestoresOptions) ([]velerov1.Restore, error) {
	bsl, err := findBackupStoreLocation()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get velero namespace")
	}

	veleroNamespace := bsl.Namespace

	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	r, err := veleroClient.Restores(veleroNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list restores")
	}

	restores := []velerov1.Restore{}

	for _, restore := range r.Items {
		if restore.Annotations["kots.io/instance"] != "true" {
			continue
		}

		if options.Namespace != "" && restore.Annotations["kots.io/kotsadm-deploy-namespace"] != options.Namespace {
			continue
		}

		restores = append(restores, restore)
	}

	return restores, nil
}

func initiateKotsadmApplicationsRestore(backupName string, kotsadmNamespace string, kotsadmPodName string, kubernetesConfigFlags *genericclioptions.ConfigFlags, log *logger.Logger) error {
	stopCh := make(chan struct{})
	defer close(stopCh)

	localPort, errChan, err := k8sutil.PortForward(kubernetesConfigFlags, 0, 3000, kotsadmNamespace, kotsadmPodName, false, stopCh, log)
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

	authSlug, err := auth.GetOrCreateAuthSlug(kubernetesConfigFlags, kotsadmNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to get kotsadm auth slug")
	}

	url := fmt.Sprintf("http://localhost:%d/api/v1/snapshot/%s/restore-apps", localPort, backupName)

	newRequest, err := http.NewRequest("POST", url, nil)
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

func waitForKotsadmApplicationsRestore(kotsadmNamespace string, kotsadmPodName string, kubernetesConfigFlags *genericclioptions.ConfigFlags, log *logger.Logger) error {
	stopCh := make(chan struct{})
	defer close(stopCh)

	localPort, errChan, err := k8sutil.PortForward(kubernetesConfigFlags, 0, 3000, kotsadmNamespace, kotsadmPodName, false, stopCh, log)
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

	authSlug, err := auth.GetOrCreateAuthSlug(kubernetesConfigFlags, kotsadmNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to get kotsadm auth slug")
	}

	url := fmt.Sprintf("http://localhost:%d/api/v1/snapshot/apps-restore-status", localPort)

	for {
		newRequest, err := http.NewRequest("GET", url, nil)
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

		type AppsRestoreStatusResponse struct {
			Status string `json:"status,omitempty"`
			Error  string `json:"error,omitempty"`
		}
		var appsRestoreStatusResponse AppsRestoreStatusResponse
		if err := json.Unmarshal(respBody, &appsRestoreStatusResponse); err != nil {
			return errors.Wrap(err, "failed to unmarshal response")
		}

		if appsRestoreStatusResponse.Error != "" {
			return errors.New(appsRestoreStatusResponse.Error)
		}

		if appsRestoreStatusResponse.Status != "running" {
			return nil
		}

		time.Sleep(time.Second * 2)
	}
}
