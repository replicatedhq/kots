package identity

import (
	"context"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"k8s.io/client-go/kubernetes"
)

func Deploy(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig kotsv1beta1.IdentityConfig, ingressConfig kotsv1beta1.IngressConfig, registryOptions *kotsadmtypes.KotsadmOptions) error {
	if err := ValidateConfig(ctx, namespace, identityConfig, ingressConfig); err != nil {
		return errors.Wrap(err, "invalid identity config")
	}

	dexConfig, err := getDexConfig(ctx, clientset, namespace, identityConfig.Spec, ingressConfig.Spec)
	if err != nil {
		return errors.Wrap(err, "failed to marshal dex config")
	}

	return identitydeploy.Deploy(ctx, clientset, namespace, "kotsadm", dexConfig, identityConfig.Spec.IngressConfig, registryOptions)
}

func Configure(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig kotsv1beta1.IdentityConfig, ingressConfig kotsv1beta1.IngressConfig) error {
	if err := ValidateConfig(ctx, namespace, identityConfig, ingressConfig); err != nil {
		return errors.Wrap(err, "invalid identity config")
	}

	dexConfig, err := getDexConfig(ctx, clientset, namespace, identityConfig.Spec, ingressConfig.Spec)
	if err != nil {
		return errors.Wrap(err, "failed to marshal dex config")
	}

	return identitydeploy.Configure(ctx, clientset, namespace, "kotsadm", dexConfig)
}

func Render(ctx context.Context, clientset kubernetes.Interface, namespace string, identityConfig kotsv1beta1.IdentityConfig, ingressConfig kotsv1beta1.IngressConfig, registryOptions *kotsadmtypes.KotsadmOptions) ([]byte, error) {
	dexConfig, err := getDexConfig(ctx, clientset, namespace, identityConfig.Spec, ingressConfig.Spec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal dex config")
	}

	return identitydeploy.Render(ctx, clientset, namespace, "kotsadm", dexConfig, identityConfig.Spec.IngressConfig, registryOptions)
}

func Undeploy(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	return identitydeploy.Undeploy(ctx, clientset, namespace, "kotsadm")
}
