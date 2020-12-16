package midstream

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
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

	deployOptions := identitydeploy.Options{
		NamePrefix:         options.AppSlug,
		IdentitySpec:       m.IdentitySpec.Spec,
		IdentityConfigSpec: m.IdentityConfig.Spec,
		IsOpenShift:        false, // TODO (ethan): openshift support
		ImageRewriteFn:     nil,   // TODO (ethan): do we rewrite in kustomization.images?
		ProxyEnv:           nil,   // TODO (ethan): do we need to configure proxy here?
		AdditionalLabels:   additionalLabels,
		Cipher:             &options.Cipher,
		Builder:            &options.Builder,
	}

	resources, err := identitydeploy.Render(ctx, deployOptions)
	if err != nil {
		return "", errors.Wrap(err, "failed to render identity service resources")
	}

	if m.IdentityConfig.Spec.Storage.PostgresConfig != nil {
		postgresSecretResource, err := identitydeploy.RenderPostgresSecret(ctx, options.AppSlug, &options.Cipher, *m.IdentityConfig.Spec.Storage.PostgresConfig, additionalLabels)
		if err != nil {
			return "", errors.Wrap(err, "failed to render postgres secret")
		}
		resources["postgressecret.yaml"] = postgresSecretResource
	}

	if m.IdentityConfig.Spec.ClientID != "" {
		clientSecret, err := m.IdentityConfig.Spec.ClientSecret.GetValue(options.Cipher)
		if err != nil {
			return "", errors.Wrap(err, "failed to decrypt client secret")
		}

		clientSecretResource, err := identitydeploy.RenderClientSecret(ctx, m.IdentityConfig.Spec.ClientID, clientSecret, additionalLabels)
		if err != nil {
			return "", errors.Wrap(err, "failed to render client secret")
		}
		resources["clientsecret.yaml"] = clientSecretResource
	}

	kustomization := kustomizetypes.Kustomization{
		TypeMeta: kustomizetypes.TypeMeta{
			APIVersion: "kustomize.config.k8s.io/v1beta1",
			Kind:       "Kustomization",
		},
	}

	for filename, resource := range resources {
		if err := ioutil.WriteFile(filepath.Join(absDir, filename), resource, 0644); err != nil {
			return "", errors.Wrapf(err, "failed to write resource %s", filename)
		}
		kustomization.Resources = append(kustomization.Resources, filename)
	}

	if err := k8sutil.WriteKustomizationToFile(kustomization, filepath.Join(absDir, "kustomization.yaml")); err != nil {
		return "", errors.Wrap(err, "failed to write kustomization file")
	}

	return base, nil
}
