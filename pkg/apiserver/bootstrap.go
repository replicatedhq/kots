package apiserver

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	identity "github.com/replicatedhq/kots/pkg/kotsadmidentity"
	identitystore "github.com/replicatedhq/kots/pkg/kotsadmidentity/store"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	"k8s.io/client-go/kubernetes/scheme"
)

type BootstrapParams struct {
	AutoCreateClusterToken string
}

func bootstrap(params BootstrapParams) error {
	if err := store.GetStore().Init(); err != nil {
		return errors.Wrap(err, "failed to init store")
	}

	if !util.IsHelmManaged() {
		if err := bootstrapClusterToken(params.AutoCreateClusterToken); err != nil {
			return errors.Wrap(err, "failed to bootstrap cluster token")
		}
	}

	return nil
}

func bootstrapClusterToken(autoCreateClusterToken string) error {
	if autoCreateClusterToken == "" {
		return errors.New("autoCreateClusterToken is not set")
	}

	_, err := store.GetStore().GetClusterIDFromDeployToken(autoCreateClusterToken)
	if err == nil {
		return nil
	}

	if err != nil && !store.GetStore().IsNotFound(err) {
		return errors.Wrap(err, "failed to lookup cluster ID")
	}

	_, err = store.GetStore().CreateNewCluster("", true, "this-cluster", autoCreateClusterToken)
	if err != nil {
		return errors.Wrap(err, "failed to create cluster")
	}

	return nil
}

func bootstrapIdentity() error {
	err := identitystore.GetStore().CreateDexDatabase("dex", "dex", os.Getenv("DEX_PGPASSWORD"))
	if err != nil {
		return errors.Wrap(err, "failed to create identity db")
	}

	// After snapshot restore, we need to create dex db for each app.
	// But the password has to match the one in the app's secret.
	// The secret is restored after kotsadm comes up, but we can get it from
	// the app's archive files.
	apps, err := store.GetStore().ListInstalledApps()
	if err != nil {
		return errors.Wrap(err, "failed to list apps")
	}

	for _, app := range apps {
		needsBootstrap, err := identity.AppIdentityNeedsBootstrap(app.Slug)
		if err != nil {
			return errors.Wrapf(err, "failed to check identity needs bootstrap for app %s", app.Slug)
		}

		if !needsBootstrap {
			continue
		}

		latestSequence, err := store.GetStore().GetLatestAppSequence(app.ID, true)
		if err != nil {
			return errors.Wrap(err, "failed to get latest app sequence")
		}

		currentArchivePath, err := ioutil.TempDir("", "kotsadm")
		if err != nil {
			return errors.Wrap(err, "failed to create temp dir")
		}
		defer os.RemoveAll(currentArchivePath)

		err = store.GetStore().GetAppVersionArchive(app.ID, latestSequence, currentArchivePath)
		if err != nil {
			return errors.Wrap(err, "failed to get current archive")
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode

		identityConfigFile := filepath.Join(currentArchivePath, "upstream", "userdata", "identityconfig.yaml")
		identityConfigData, err := ioutil.ReadFile(identityConfigFile)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return errors.Wrapf(err, "failed to get stat identity config file for app %s", app.Slug)
		}

		obj, gvk, err := decode([]byte(identityConfigData), nil, nil)
		if err != nil {
			return errors.Wrap(err, "failed to decode config data")
		}
		if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "IdentityConfig" {
			return errors.Errorf("expected IdentityConfig, but found %s/%s/%s", gvk.Group, gvk.Version, gvk.Kind)
		}
		identityConfig := obj.(*kotsv1beta1.IdentityConfig)

		identityConfigFile, err = identity.InitAppIdentityConfig(app.Slug, identityConfig.Spec.Storage)
		if err != nil {
			return errors.Wrap(err, "failed to init identity config")
		}
		// don't need the temp file. it should be identical to the one loaded from userdata
		_ = os.Remove(identityConfigFile)
	}

	return nil
}
