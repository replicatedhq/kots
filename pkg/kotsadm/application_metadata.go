package kotsadm

import (
	"bytes"

	"github.com/pkg/errors"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getApplicationMetadataYAML(data []byte, namespace string) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var configMap bytes.Buffer
	if err := s.Encode(applicationMetadataConfig(data, namespace), &configMap); err != nil {
		return nil, errors.Wrap(err, "failed to marshal minio config map")
	}
	docs["application.yaml"] = configMap.Bytes()

	return docs, nil
}

func ensureApplicationMetadata(deployOptions DeployOptions, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Get("kotsadm-application-metadata", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing metadata config map")
		}

		_, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Create(applicationMetadataConfig(deployOptions.ApplicationMetadata, deployOptions.Namespace))
		if err != nil {
			return errors.Wrap(err, "failed to create metadata config map")
		}
	}

	return nil
}
