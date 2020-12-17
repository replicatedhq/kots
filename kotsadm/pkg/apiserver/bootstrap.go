package apiserver

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/identity"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
	"k8s.io/client-go/kubernetes/scheme"
)

func bootstrap() error {
	if err := store.GetStore().Init(); err != nil {
		return errors.Wrap(err, "failed to init store")
	}

	if err := bootstrapClusterToken(); err != nil {
		return errors.Wrap(err, "failed to bootstrap cluster token")
	}

	return nil
}

func bootstrapClusterToken() error {
	if os.Getenv("AUTO_CREATE_CLUSTER_TOKEN") == "" {
		return errors.New("AUTO_CREATE_CLUSTER_TOKEN is not set")
	}

	_, err := store.GetStore().GetClusterIDFromDeployToken(os.Getenv("AUTO_CREATE_CLUSTER_TOKEN"))
	if err == nil {
		return nil
	}

	if err != nil && !store.GetStore().IsNotFound(err) {
		return errors.Wrap(err, "failed to lookup cluster ID")
	}

	_, err = store.GetStore().CreateNewCluster("", true, "this-cluster", os.Getenv("AUTO_CREATE_CLUSTER_TOKEN"))
	if err != nil {
		return errors.Wrap(err, "failed to create cluster")
	}

	return nil
}

// After snapshot restore, we need to create dex db for each app.
// But the password has to match the one in the app's secret.
// The secret is restored after kotsadm comes up, but we can get it from
// the app's archive files.
func bootstrapIdentity() error {
	apps, err := store.GetStore().ListInstalledApps()
	if err != nil {
		return errors.Wrap(err, "failed to list apps")
	}

	apiCipher, err := crypto.AESCipherFromString(os.Getenv("API_ENCRYPTION_KEY"))
	if err != nil {
		return errors.Wrap(err, "failed to create aes cipher")
	}

	for _, app := range apps {
		needsBootstrap, err := identity.AppIdentityNeedsBootstrap(app.Slug)
		if err != nil {
			return errors.Wrapf(err, "failed to check identity needs bootstrap for app %s", app.Slug)
		}

		if !needsBootstrap {
			continue
		}

		currentArchivePath, err := ioutil.TempDir("", "kotsadm")
		if err != nil {
			return errors.Wrap(err, "failed to create temp dir")
		}
		defer os.RemoveAll(currentArchivePath)

		err = store.GetStore().GetAppVersionArchive(app.ID, app.CurrentSequence, currentArchivePath)
		if err != nil {
			return errors.Wrap(err, "failed to get current archive")
		}

		identityConfigFile := filepath.Join(currentArchivePath, "upstream", "userdata", "identityconfig.yaml")
		identityConfigData, err := ioutil.ReadFile(identityConfigFile)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return errors.Wrapf(err, "failed to get stat identity config file for app %s", app.Slug)
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode([]byte(identityConfigData), nil, nil)
		if err != nil {
			return errors.Wrap(err, "failed to decode config data")
		}
		if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "IdentityConfig" {
			return errors.Errorf("expected IdentityConfig, but found %s/%s/%s", gvk.Group, gvk.Version, gvk.Kind)
		}
		identityConfig := obj.(*kotsv1beta1.IdentityConfig)

		identityConfigFile, err = identity.InitAppIdentityConfig(app.Slug, identityConfig.Spec.Storage, *apiCipher)
		if err != nil {
			return errors.Wrap(err, "failed to init identity config")
		}
		// don't need the temp file. it should be identical to the one loaded from userdata
		_ = os.Remove(identityConfigFile)
	}

	return nil
}
