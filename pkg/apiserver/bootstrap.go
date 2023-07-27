package apiserver

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
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
		if err := loadEncryptionKeys(); err != nil {
			return errors.Wrap(err, "failed to load encryption keys")
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

func loadEncryptionKeys() error {
	apps, err := store.GetStore().ListInstalledApps()
	if err != nil {
		return errors.Wrap(err, "failed to list apps")
	}

	for _, app := range apps {
		latestSequence, err := store.GetStore().GetLatestAppSequence(app.ID, true)
		if err != nil {
			return errors.Wrap(err, "failed to get latest app sequence")
		}

		currentArchivePath, err := os.MkdirTemp("", "kotsadm")
		if err != nil {
			return errors.Wrap(err, "failed to create temp dir")
		}
		defer os.RemoveAll(currentArchivePath)

		err = store.GetStore().GetAppVersionArchive(app.ID, latestSequence, currentArchivePath)
		if err != nil {
			return errors.Wrap(err, "failed to get current archive")
		}

		installation, err := kotsutil.LoadInstallationFromPath(filepath.Join(currentArchivePath, "upstream", "userdata", "installation.yaml"))
		if err != nil {
			return errors.Wrap(err, "failed to load installation from path")
		}

		// add installation encryption key to list of decryption ciphers
		err = crypto.InitFromString(installation.Spec.EncryptionKey)
		if err != nil {
			return errors.Wrap(err, "failed to load encryption cipher")
		}
	}

	return nil
}
