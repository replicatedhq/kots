package snapshot

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
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
	log.ActionWithSpinner("Deleting Admin Console Objects")

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
	log.ActionWithSpinner("Creating a Restore")

	trueVal := true
	restore := &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: veleroNamespace,
			Name:      options.BackupName, // restore name same as backup name
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
					kotsadmtypes.BackupLabel: kotsadmtypes.BackupLabelValue,
				},
			},
			RestorePVs:              &trueVal,
			IncludeClusterResources: &trueVal,
		},
	}

	// delete existing restore object (if exists)
	err = veleroClient.Restores(veleroNamespace).Delete(context.TODO(), options.BackupName, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		log.FinishSpinnerWithError()
		return nil, errors.Wrapf(err, "failed to delete restore %s", options.BackupName)
	}

	// create new restore object
	restore, err = veleroClient.Restores(veleroNamespace).Create(context.TODO(), restore, metav1.CreateOptions{})
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to create restore")
	}

	log.FinishSpinner()
	log.ActionWithoutSpinner(fmt.Sprintf("Restore request has been created. Restore name is %s", restore.ObjectMeta.Name))

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
