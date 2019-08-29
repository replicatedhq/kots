package kotsadm

import (
	"bytes"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
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
