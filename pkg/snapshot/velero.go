package snapshot

import (
	"github.com/replicatedhq/kots/pkg/kotsadm"

	"context"

	"github.com/pkg/errors"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func CreateRBACPermissions(kotsadmNamespace string) error {
	veleroNamespace, err := DetectVeleroNamespace()
	if err != nil {
		return errors.Wrap(err, "failed to detect velero namespace")
	}
	if veleroNamespace == "" {
		return errors.New("velero not found")
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create kubernetes clientset")
	}

	if err := kotsadm.EnsureKotsadmRole(veleroNamespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm role")
	}

	if err := kotsadm.EnsureKotsadmRoleBinding(veleroNamespace, kotsadmNamespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm rolebinding")
	}

	return nil
}

func DetectVeleroNamespace() (string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "", errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to create velero clientset")
	}

	backupStorageLocations, err := veleroClient.BackupStorageLocations("").List(context.TODO(), metav1.ListOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return "", nil
	}

	if err != nil {
		// can't detect velero
		return "", nil
	}

	for _, backupStorageLocation := range backupStorageLocations.Items {
		if backupStorageLocation.Name == "default" {
			return backupStorageLocation.Namespace, nil
		}
	}

	return "", nil
}
