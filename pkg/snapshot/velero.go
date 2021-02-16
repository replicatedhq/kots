package snapshot

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/kotsadm"

	"context"

	"github.com/pkg/errors"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func EnsureVeleroPermissions(veleroNamespace string, kotsadmNamespace string) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create velero clientset")
	}

	backupStorageLocations, err := veleroClient.BackupStorageLocations(veleroNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to list backupstoragelocations in '%s' namespace", veleroNamespace)
	}

	verifiedVeleroNamespace := ""
	for _, backupStorageLocation := range backupStorageLocations.Items {
		if backupStorageLocation.Name == "default" {
			verifiedVeleroNamespace = backupStorageLocation.Namespace
			break
		}
	}

	if verifiedVeleroNamespace == "" {
		return errors.New(fmt.Sprintf("could not detect velero in '%s' namespace", veleroNamespace))
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create kubernetes clientset")
	}

	if err := kotsadm.EnsureKotsadmRole(verifiedVeleroNamespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm role")
	}

	if err := kotsadm.EnsureKotsadmRoleBinding(verifiedVeleroNamespace, kotsadmNamespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm rolebinding")
	}

	return nil
}

func EnsureVeleroNamespaceConfigMap(veleroNamespace string, kotsadmNamespace string) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create kubernetes clientset")
	}

	existingConfigMap, err := clientset.CoreV1().ConfigMaps(kotsadmNamespace).Get(context.TODO(), "kotsadm-velero-namespace", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to lookup velero configmap")
		}

		newConfigMap := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kotsadm-velero-namespace",
				Namespace: kotsadmNamespace,
				Labels:    kotsadmtypes.GetKotsadmLabels(),
			},
			Data: map[string]string{
				"veleroNamespace": veleroNamespace,
			},
		}

		_, err := clientset.CoreV1().ConfigMaps(kotsadmNamespace).Create(context.TODO(), newConfigMap, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create velero configmap")
		}

		return nil
	}

	existingConfigMap.Data["veleroNamespace"] = veleroNamespace

	_, err = clientset.CoreV1().ConfigMaps(kotsadmNamespace).Update(context.TODO(), existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update velero configmap")
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
