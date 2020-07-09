package kotsadm

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getConfigMapsYAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var configMap bytes.Buffer
	if err := s.Encode(kotsadmConfigMap(deployOptions), &configMap); err != nil {
		return nil, errors.Wrap(err, "failed to marshal minio config map")
	}
	docs["kotsadm-config.yaml"] = configMap.Bytes()

	return docs, nil
}

func ensureKotsadmConfig(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	if err := ensurePrivateKotsadmRegistrySecret(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure private kotsadm registry secret")
	}

	if err := ensureConfigMaps(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure private kotsadm registry secret")
	}

	return nil
}

func ensureConfigMaps(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Get(context.TODO(), types.KotsadmConfigMap, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing kotsadm config map")
		}

		_, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Create(context.TODO(), kotsadmConfigMap(deployOptions), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create kotsadm config map")
		}
	}

	return nil
}
