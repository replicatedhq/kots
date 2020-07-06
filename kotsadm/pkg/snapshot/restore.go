package snapshot

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	veleroapiv1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func CreateRestore(snapshotName string) error {
	// Reference https://github.com/vmware-tanzu/velero/blob/42b612645863c2b3e451b447f9bf798295dd7dba/pkg/cmd/cli/restore/create.go#L222

	logger.Debug("creating restore",
		zap.String("snapshotName", snapshotName))

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

	_, err = veleroClient.Backups(veleroNamespace).Get(context.TODO(), snapshotName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to find backup")
	}

	trueVal := true
	restore := &veleroapiv1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: veleroNamespace,
			Name:      snapshotName, // restore name same as snapshot name
			// Labels:    o.Labels.Data(),
		},
		Spec: veleroapiv1.RestoreSpec{
			BackupName: snapshotName,
			// ScheduleName: o.ScheduleName,
			// IncludedNamespaces:      o.IncludeNamespaces,
			// ExcludedNamespaces:      o.ExcludeNamespaces,
			// IncludedResources:       o.IncludeResources,
			// ExcludedResources:       o.ExcludeResources,
			// NamespaceMapping:        o.NamespaceMappings.Data(),
			// LabelSelector:           o.Selector.LabelSelector,
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

func DeleteRestore(snapshotName string) error {
	bsl, err := findBackupStoreLocation()
	if err != nil {
		return errors.Wrap(err, "failed to get velero namespace")
	}

	veleroNamespace := bsl.Namespace

	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create clientset")
	}

	err = veleroClient.Restores(veleroNamespace).Delete(context.TODO(), snapshotName, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return errors.Wrapf(err, "failed to delete restore %s", snapshotName)
	}

	return nil
}

func GetBackup(snapshotName string) (*veleroapiv1.Backup, error) {
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
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	backup, err := veleroClient.Backups(veleroNamespace).Get(context.TODO(), snapshotName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get backup")
	}

	return backup, nil
}
