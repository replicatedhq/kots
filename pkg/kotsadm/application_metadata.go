package kotsadm

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getApplicationMetadataYAML(data []byte, namespace string, upstreamURI string) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var configMap bytes.Buffer
	if err := s.Encode(kotsadmobjects.ApplicationMetadataConfig(data, namespace, upstreamURI), &configMap); err != nil {
		return nil, errors.Wrap(err, "failed to marshal metadata config map")
	}
	docs["application.yaml"] = configMap.Bytes()

	return docs, nil
}

func ensureApplicationMetadata(deployOptions types.DeployOptions, clientset kubernetes.Interface) error {
	_, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Get(context.TODO(), "kotsadm-application-metadata", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing metadata config map")
		}

		_, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Create(context.TODO(), kotsadmobjects.ApplicationMetadataConfig(deployOptions.ApplicationMetadata, deployOptions.Namespace, deployOptions.UpstreamURI), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create metadata config map")
		}
	}

	return nil
}
