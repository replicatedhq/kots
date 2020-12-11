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
	if m.IdentitySpec == nil || m.IdentityConfig == nil {
		return "", nil
	}

	base := "identity-service"

	absDir := filepath.Join(options.MidstreamDir, base)
	if err := os.MkdirAll(absDir, 0744); err != nil {
		return "", errors.Wrap(err, "failed to mkdir")
	}

	// TODO (ethan): customize labels (dont use kustomize)
	deployOptions := identitydeploy.Options{
		NamePrefix:         options.AppSlug,
		IdentitySpec:       m.IdentitySpec.Spec,
		IdentityConfigSpec: m.IdentityConfig.Spec,
		ImageRewriteFn:     nil, // TODO (ethan): do we rewrite in kustomization.images?
	}
	resources, err := identitydeploy.Render(ctx, deployOptions)
	if err != nil {
		return "", errors.Wrap(err, "failed to render identity service")
	}

	// support for the dev environment where app is in "test" namespace
	host := "kotsadm-postgres"
	if kotsadmNamespace := os.Getenv("POD_NAMESPACE"); kotsadmNamespace != "" {
		host = fmt.Sprintf("%s.%s", host, kotsadmNamespace)
	}

	if _, err = os.Stat(filepath.Join(absDir, "postgressecret.yaml")); os.IsNotExist(err) {
		postgresSecret, err := identitydeploy.RenderPostgresSecret(ctx, options.AppSlug, m.IdentityConfig.Spec.Storage.PostgresConfig)
		if err != nil {
			return "", errors.Wrap(err, "failed to render postgres secret")
		}
		resources["postgressecret.yaml"] = postgresSecret
	} else if err != nil {
		return "", errors.Wrap(err, "failed to stat postgres secret")
	}

	if _, err = os.Stat(filepath.Join(absDir, "clientsecret.yaml")); os.IsNotExist(err) {
		clientSecret, err := identitydeploy.RenderClientSecret(ctx, options.AppSlug, m.IdentityConfig.Spec.ClientSecret)
		if err != nil {
			return "", errors.Wrap(err, "failed to render client secret")
		}
		resources["clientsecret.yaml"] = clientSecret
	} else if err != nil {
		return "", errors.Wrap(err, "failed to stat client secret")
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
