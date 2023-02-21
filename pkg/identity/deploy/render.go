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

	// TODO (salah): make this work with minimal rbac
	ns := namespaceResource(options)
	buf := bytes.NewBuffer(nil)
	if err := s.Encode(ns, buf); err != nil {
		return nil, errors.Wrap(err, "failed to encode namespace")
	}
	resources["namespace.yaml"] = buf.Bytes()

	clusterRole := clusterRoleResource(options)
	buf = bytes.NewBuffer(nil)
	if err := s.Encode(clusterRole, buf); err != nil {
		return nil, errors.Wrap(err, "failed to encode cluster role")
	}
	resources["clusterrole.yaml"] = buf.Bytes()

	serviceAccount := serviceAccountResource(options)
	buf = bytes.NewBuffer(nil)
	if err := s.Encode(serviceAccount, buf); err != nil {
		return nil, errors.Wrap(err, "failed to encode service account")
	}
	resources["serviceaccount.yaml"] = buf.Bytes()

	clusterRoleBinding := clusterRoleBindingResource(options)
	buf = bytes.NewBuffer(nil)
	if err := s.Encode(clusterRoleBinding, buf); err != nil {
		return nil, errors.Wrap(err, "failed to encode cluster role binding")
	}
	resources["clusterrolebinding.yaml"] = buf.Bytes()

	secret := secretResource(dexConfig, options)
	buf = bytes.NewBuffer(nil)
	if err := s.Encode(secret, buf); err != nil {
		return nil, errors.Wrap(err, "failed to encode secret")
	}
	resources["secret.yaml"] = buf.Bytes()

	if options.IdentitySpec.WebConfig != nil && options.IdentitySpec.WebConfig.Theme != nil {
		configMap, err := dexThemeConfigMapResource(options)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get dex theme config map resource")
		}
		buf = bytes.NewBuffer(nil)
		if err := s.Encode(configMap, buf); err != nil {
			return nil, errors.Wrap(err, "failed to encode dex theme config map resource")
		}
		resources["dexthemeconfigmap.yaml"] = buf.Bytes()
	}

	deployment, err := deploymentResource(issuerURL, configChecksum, options)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deployment resource")
	}
	buf = bytes.NewBuffer(nil)
	if err := s.Encode(deployment, buf); err != nil {
		return nil, errors.Wrap(err, "failed to encode deployment")
	}
	resources["deployment.yaml"] = buf.Bytes()

	service := serviceResource(options)
	buf = bytes.NewBuffer(nil)
	if err := s.Encode(service, buf); err != nil {
		return nil, errors.Wrap(err, "failed to encode service")
	}
	resources["service.yaml"] = buf.Bytes()

	if options.IdentityConfigSpec.IngressConfig.Enabled {
		if ingressConfig := options.IdentityConfigSpec.IngressConfig.Ingress; ingressConfig != nil {
			ingress := ingressResource(options)
			buf = bytes.NewBuffer(nil)
			if err := s.Encode(ingress, buf); err != nil {
				return nil, errors.Wrap(err, "failed to encode ingress")
			}
			resources["ingress.yaml"] = buf.Bytes()
		}
	}

	if options.IdentityConfigSpec.ClientID != "" {
		clientSecret, err := options.IdentityConfigSpec.ClientSecret.GetValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to decrypt client secret")
		}
		clientSecretResource, err := renderClientSecret(ctx, options.Namespace, options.IdentityConfigSpec.ClientID, clientSecret, options.AdditionalLabels)
		if err != nil {
			return nil, errors.Wrap(err, "failed to render client secret")
		}
		resources["clientsecret.yaml"] = clientSecretResource
	}

	return resources, nil
}
