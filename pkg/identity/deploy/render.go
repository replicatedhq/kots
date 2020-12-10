package deploy

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func Render(ctx context.Context, namePrefix string, dexConfig []byte, ingressSpec kotsv1beta1.IngressConfigSpec, imageRewriteFn ImageRewriteFunc) (map[string][]byte, error) {
	configChecksum := fmt.Sprintf("%x", md5.Sum(dexConfig))

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	resources := map[string][]byte{}

	// TODO (ethan): postgres

	secret := secretResource(namePrefix, dexConfig)
	buf := bytes.NewBuffer(nil)
	if err := s.Encode(secret, buf); err != nil {
		return nil, errors.Wrap(err, "failed to encode secret")
	}
	resources["secret.yaml"] = buf.Bytes()

	deployment := deploymentResource(namePrefix, configChecksum, imageRewriteFn)
	buf = bytes.NewBuffer(nil)
	if err := s.Encode(deployment, buf); err != nil {
		return nil, errors.Wrap(err, "failed to encode deployment")
	}
	resources["deployment.yaml"] = buf.Bytes()

	service := serviceResource(namePrefix, ingressSpec)
	buf = bytes.NewBuffer(nil)
	if err := s.Encode(service, buf); err != nil {
		return nil, errors.Wrap(err, "failed to encode service")
	}
	resources["service.yaml"] = buf.Bytes()

	if ingressSpec.Ingress != nil {
		ingress := ingressResource(namePrefix, *ingressSpec.Ingress)
		buf = bytes.NewBuffer(nil)
		if err := s.Encode(ingress, buf); err != nil {
			return nil, errors.Wrap(err, "failed to encode ingress")
		}
		resources["ingress.yaml"] = buf.Bytes()
	}

	return resources, nil
}
