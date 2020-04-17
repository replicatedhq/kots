package snapshot

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func CreateRestore(snapshotName string) error {
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

	backup, err := veleroClient.Backups(veleroNamespace).Get(snapshotName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create velero backup")
	}

	fmt.Printf("%#v", backup)
	return nil
}
