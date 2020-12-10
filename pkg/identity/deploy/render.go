package deploy

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func Render(ctx context.Context, clientset kubernetes.Interface, namespace string, namePrefix string, dexConfig []byte, ingressSpec kotsv1beta1.IngressConfigSpec, registryOptions *kotsadmtypes.KotsadmOptions) ([]byte, error) {
	configChecksum := fmt.Sprintf("%x", md5.Sum(dexConfig))

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	buf := bytes.NewBuffer(nil)

	// TODO: postgres

	secret := secretResource(namePrefix, dexConfig)
	if err := s.Encode(secret, buf); err != nil {
		return nil, errors.Wrap(err, "failed to encode secret")
	}

	deployment := deploymentResource(namePrefix, configChecksum, namespace, registryOptions)
	if err := s.Encode(deployment, buf); err != nil {
		return nil, errors.Wrap(err, "failed to encode deployment")
	}

	service := serviceResource(namePrefix, ingressSpec)
	if err := s.Encode(service, buf); err != nil {
		return nil, errors.Wrap(err, "failed to encode service")
	}

	if ingressSpec.Ingress != nil {
		ingress := ingressResource(namespace, namePrefix, *ingressSpec.Ingress)
		if err := s.Encode(ingress, buf); err != nil {
			return nil, errors.Wrap(err, "failed to encode ingress")
		}
	}

	return buf.Bytes(), nil
}
