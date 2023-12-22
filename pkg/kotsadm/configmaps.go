package kotsadm

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	kotsadmresources "github.com/replicatedhq/kots/pkg/kotsadm/resources"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	MinioXlMigrationStatusConfigmapName = "kotsadm-minio-xl-migration-status"
)

func getConfigMapsYAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var kotsadmConfigMap bytes.Buffer
	if err := s.Encode(kotsadmobjects.KotsadmConfigMap(deployOptions), &kotsadmConfigMap); err != nil {
		return nil, errors.Wrap(err, "failed to marshal kotsadm config map")
	}
	docs["kotsadm-config.yaml"] = kotsadmConfigMap.Bytes()

	return docs, nil
}

func ensureKotsadmConfig(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	if err := kotsadmresources.EnsurePrivateKotsadmRegistrySecret(deployOptions.Namespace, deployOptions.RegistryConfig, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure private kotsadm registry secret")
	}

	if err := EnsureConfigMaps(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm config maps")
	}

	return nil
}

func EnsureConfigMaps(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	desiredConfigMap := kotsadmobjects.KotsadmConfigMap(deployOptions)

	existingConfigMap, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Get(context.TODO(), types.KotsadmConfigMap, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing kotsadm config map")
		}

		_, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Create(context.TODO(), desiredConfigMap, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create kotsadm config map")
		}

		return nil
	}

	existingConfigMap = updateConfigMap(existingConfigMap, desiredConfigMap)

	_, err = clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Update(context.TODO(), existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update kotsadm config map")
	}

	return nil
}

func updateConfigMap(existingConfigMap, desiredConfigMap *corev1.ConfigMap) *corev1.ConfigMap {
	existingConfigMap.Data = desiredConfigMap.Data
	return existingConfigMap
}

func ensureWaitForAirgapConfig(deployOptions types.DeployOptions, clientset *kubernetes.Clientset, configMapName string) error {
	additionalLabels := map[string]string{
		"kots.io/automation": "airgap",
	}
	if deployOptions.License != nil {
		additionalLabels["kots.io/app"] = deployOptions.License.Spec.AppSlug
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: deployOptions.Namespace,
			Labels:    types.GetKotsadmLabels(additionalLabels),
		},
		Data: map[string]string{
			"wait-for-airgap-app": "true",
		},
	}

	_, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
	if err != nil {
		if !kuberneteserrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "failed to create kotsadm config map")
		}
	} else {
		return nil
	}

	_, err = clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update kotsadm config map")
	}

	return nil
}

func ensureConfigFromFile(deployOptions types.DeployOptions, clientset *kubernetes.Clientset, configMapName string, filename string) error {
	configMap, err := configMapFromFile(deployOptions, configMapName, filename)
	if err != nil {
		return errors.Wrap(err, "failed to build config map")
	}

	_, err = clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
	if err != nil {
		if !kuberneteserrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "failed to create kotsadm config map")
		}
	} else {
		return nil
	}

	_, err = clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update kotsadm config map")
	}

	return nil
}

func configMapFromFile(deployOptions types.DeployOptions, configMapName string, filename string) (*corev1.ConfigMap, error) {
	fileData, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load file")
	}

	key := filepath.Base(filename)
	value := base64.StdEncoding.EncodeToString(fileData)

	data := map[string]string{
		key: value,
	}

	additionalLabels := map[string]string{
		"kots.io/automation": "airgap",
	}
	if deployOptions.License != nil {
		additionalLabels["kots.io/app"] = deployOptions.License.Spec.AppSlug
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: deployOptions.Namespace,
			Labels:    types.GetKotsadmLabels(additionalLabels),
		},
		Data: data,
	}

	return configMap, nil
}

func ensureMinioXlMigrationScriptsConfigmap(namespace string, clientset kubernetes.Interface) error {
	desiredConfigMap := kotsadmobjects.MinioXlMigrationScriptsConfigMap(namespace)

	existingConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), desiredConfigMap.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing minio xl migration scripts configmap")
		}

		_, err := clientset.CoreV1().ConfigMaps(namespace).Create(context.TODO(), desiredConfigMap, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create minio xl migration scripts configmap")
		}

		return nil
	}

	existingConfigMap = updateConfigMap(existingConfigMap, desiredConfigMap)

	_, err = clientset.CoreV1().ConfigMaps(namespace).Update(context.TODO(), existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update minio xl migration scripts configmap")
	}

	return nil
}

func ensureMinioXlMigrationStatusConfigmap(namespace string, clientset kubernetes.Interface) error {
	_, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), MinioXlMigrationStatusConfigmapName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing minio xl migration status configmap")
		}

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: MinioXlMigrationStatusConfigmapName,
			},
			Data: map[string]string{
				"status": "running",
			},
		}

		_, err := clientset.CoreV1().ConfigMaps(namespace).Create(context.TODO(), cm, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create minio xl migration status configmap")
		}
	}

	return nil
}
