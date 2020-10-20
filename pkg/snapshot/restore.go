package snapshot

import (
	"context"

	"github.com/pkg/errors"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type InstanceRestoreOptions struct {
	BackupName string
	Silent     bool
}

func InstanceRestore(instanceRestoreOptions InstanceRestoreOptions) error {
	bsl, err := findBackupStoreLocation()
	if err != nil {
		return errors.Wrap(err, "failed to get velero namespace")
	}

	veleroNamespace := bsl.Namespace

	// get the backup
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create clientset")
	}

	_, err = veleroClient.Backups(veleroNamespace).Get(context.TODO(), instanceRestoreOptions.BackupName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to find backup")
	}

	trueVal := true
	restore := &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: veleroNamespace,
			Name:      instanceRestoreOptions.BackupName, // restore name same as backup name
		},
		Spec: velerov1.RestoreSpec{
			BackupName: instanceRestoreOptions.BackupName,
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					kotsadmtypes.BackupLabel: kotsadmtypes.BackupLabelValue,
				},
			},
			RestorePVs:              &trueVal,
			IncludeClusterResources: &trueVal,
		},
	}

	_, err = veleroClient.Restores(veleroNamespace).Create(context.TODO(), restore, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create restore")
	}

	return nil
}

func findBackupStoreLocation() (*velerov1.BackupStorageLocation, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create velero clientset")
	}

	backupStorageLocations, err := veleroClient.BackupStorageLocations("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list backupstoragelocations")
	}

	for _, backupStorageLocation := range backupStorageLocations.Items {
		if backupStorageLocation.Name == "default" {
			return &backupStorageLocation, nil
		}
	}

	return nil, errors.New("global config not found")
}
