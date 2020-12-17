package deploy

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"

	"github.com/pkg/errors"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func Render(ctx context.Context, options Options) (map[string][]byte, error) {
	issuerURL, err := dexIssuerURL(options.IdentitySpec, options.Builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get dex issuer url")
	}

	dexConfig, err := getDexConfig(ctx, issuerURL, options)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get dex config")
	}

	configChecksum := fmt.Sprintf("%x", md5.Sum(dexConfig))

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	resources := map[string][]byte{}

	secret := secretResource(options.NamePrefix, dexConfig)
	buf := bytes.NewBuffer(nil)
	if err := s.Encode(secret, buf); err != nil {
		return nil, errors.Wrap(err, "failed to encode secret")
	}
	resources["secret.yaml"] = buf.Bytes()

	deployment, err := deploymentResource(issuerURL, configChecksum, options)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deployment resource")
	}
	buf = bytes.NewBuffer(nil)
	if err := s.Encode(deployment, buf); err != nil {
		return nil, errors.Wrap(err, "failed to encode deployment")
	}
	resources["deployment.yaml"] = buf.Bytes()

	service := serviceResource(options.NamePrefix, options.IdentityConfigSpec.IngressConfig)
	buf = bytes.NewBuffer(nil)
	if err := s.Encode(service, buf); err != nil {
		return nil, errors.Wrap(err, "failed to encode service")
	}
	resources["service.yaml"] = buf.Bytes()

	if options.IdentityConfigSpec.IngressConfig.Enabled {
		if ingressConfig := options.IdentityConfigSpec.IngressConfig.Ingress; ingressConfig != nil {
			ingress := ingressResource(options.NamePrefix, *ingressConfig)
			buf = bytes.NewBuffer(nil)
			if err := s.Encode(ingress, buf); err != nil {
				return nil, errors.Wrap(err, "failed to encode ingress")
			}
			resources["ingress.yaml"] = buf.Bytes()
		}
	}

	return resources, nil
}
