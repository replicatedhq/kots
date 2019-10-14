package kotsadm

import (
	"bytes"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func GetTitledYAML(licenseData []byte, license *kotsv1beta1.License) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var config bytes.Buffer
	if err := s.Encode(titledConfigMap(licenseData), &config); err != nil {
		return nil, errors.Wrap(err, "failed to marshal titled config")
	}
	docs["titled-config.yaml"] = config.Bytes()

	var deployment bytes.Buffer
	if err := s.Encode(titledDeployment(), &deployment); err != nil {
		return nil, errors.Wrap(err, "failed to marshal titled deployment")
	}
	docs["titled-deployment.yaml"] = deployment.Bytes()

	var service bytes.Buffer
	if err := s.Encode(titledService(), &service); err != nil {
		return nil, errors.Wrap(err, "failed to marshal titled service")
	}
	docs["titled-service.yaml"] = service.Bytes()

	return docs, nil
}
