package midstream

import (
	"context"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
	identitytypes "github.com/replicatedhq/kots/pkg/identity/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

func (m *Midstream) writeIdentityService(ctx context.Context, options WriteOptions) (string, error) {
	if !identitydeploy.IsEnabled(m.IdentitySpec, m.IdentityConfig) {
		return "", nil
	}

	base := "identity-service"

	absDir := filepath.Join(options.MidstreamDir, base)
	if err := os.MkdirAll(absDir, 0744); err != nil {
		return "", errors.Wrap(err, "failed to mkdir")
	}

	additionalLabels := map[string]string{
		"kots.io/app": options.AppSlug,
	}

	proxyEnv := map[string]string{
		"HTTP_PROXY":  options.HTTPProxyEnvValue,
		"HTTPS_PROXY": options.HTTPSProxyEnvValue,
		"NO_PROXY":    options.NoProxyEnvValue,
	}

	deployOptions := identitydeploy.Options{
		NamePrefix:         options.AppSlug,
		Namespace:          identitytypes.Namespace(options.AppSlug),
		IdentitySpec:       m.IdentitySpec.Spec,
		IdentityConfigSpec: m.IdentityConfig.Spec,
		IsOpenShift:        options.IsOpenShift,
		ImageRewriteFn:     nil, // TODO (ethan): do we rewrite in kustomization.images?
		ProxyEnv:           proxyEnv,
		AdditionalLabels:   additionalLabels,
		Builder:            &options.Builder,
	}

	resources, err := identitydeploy.Render(ctx, deployOptions)
	if err != nil {
		return "", errors.Wrap(err, "failed to render identity service resources")
	}

	kustomization := kustomizetypes.Kustomization{
		TypeMeta: kustomizetypes.TypeMeta{
			APIVersion: "kustomize.config.k8s.io/v1beta1",
			Kind:       "Kustomization",
		},
	}

	for filename, resource := range resources {
		if err := os.WriteFile(filepath.Join(absDir, filename), resource, 0644); err != nil {
			return "", errors.Wrapf(err, "failed to write resource %s", filename)
		}
		kustomization.Resources = append(kustomization.Resources, filename)
	}

	if err := k8sutil.WriteKustomizationToFile(kustomization, filepath.Join(absDir, "kustomization.yaml")); err != nil {
		return "", errors.Wrap(err, "failed to write kustomization file")
	}

	return base, nil
}
