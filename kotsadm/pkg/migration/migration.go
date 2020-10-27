package migration

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/preflight"
	"github.com/replicatedhq/kots/kotsadm/pkg/render"
	"github.com/replicatedhq/kots/kotsadm/pkg/reporting"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
)

func RunMigrations() {
	if err := runDisasterRecoveryMigration(); err != nil {
		logger.Error(errors.Wrap(err, "failed to run disaster recovery migration"))
	}
}

func runDisasterRecoveryMigration() error {
	appsList, err := store.GetStore().ListInstalledApps()
	if err != nil {
		return errors.Wrap(err, "failed to list installed apps")
	}

	for _, a := range appsList {
		if err := enableDisasterRecoveryForApp(a); err != nil {
			logger.Error(errors.Wrapf(err, "failed to enable disaster recovery for app %s", a.Slug))
		}
	}

	return nil
}

func enableDisasterRecoveryForApp(a *apptypes.App) error {
	logger.Info(fmt.Sprintf("Running disaster recovery migration for app %s", a.Slug))

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to list downstreams")
	}
	if len(downstreams) == 0 {
		return nil
	}

	deployedVersion, err := downstream.GetCurrentVersion(a.ID, downstreams[0].ClusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get current downstream version")
	}

	if deployedVersion == nil {
		return nil
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	err = store.GetStore().GetAppVersionArchive(a.ID, deployedVersion.ParentSequence, archiveDir)
	if err != nil {
		return errors.Wrap(err, "failed to get app version archive")
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get registry settings")
	}

	// re-rendering will write the disaster recovery label transformer in the midstream
	if err := render.RenderDir(archiveDir, a, downstreams, registrySettings, reporting.GetReportingInfo(a.ID)); err != nil {
		return errors.Wrap(err, "failed to render new version")
	}

	newSequence, err := version.CreateVersion(a.ID, archiveDir, "Disaster Recovery Migration", a.CurrentSequence)
	if err != nil {
		return errors.Wrap(err, "failed to create new version")
	}

	if err := version.DeployVersion(a.ID, newSequence); err != nil {
		return errors.Wrap(err, "failed to deploy new version")
	}

	// preflights shouldn't block the deployment of the new version since the new version
	// will only have disaster recovery labels as diff and should not affect previous preflight results
	if err := preflight.Run(a.ID, newSequence, a.IsAirgap, archiveDir); err != nil {
		logger.Error(errors.Wrapf(err, "failed to run preflights in disaster recovery migration for app %s", a.Slug))
	}

	return nil
}
