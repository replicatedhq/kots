package autodeployer

import (
	"sort"
	"sync"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/version"
	cron "github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// jobs maps app ids to their cron jobs
var jobs = make(map[string]*cron.Cron)
var mtx sync.Mutex

// Start will start the auto deployer
// the frequency of those update checks are app specific and can be modified by the user
func Start() error {
	logger.Debug("starting auto deployer")

	appsList, err := store.GetStore().ListInstalledApps()
	if err != nil {
		return errors.Wrap(err, "failed to list installed apps")
	}

	for _, a := range appsList {
		if a.IsAirgap {
			continue
		}
		if err := Configure(a.ID); err != nil {
			logger.Error(errors.Wrapf(err, "failed to configure app %s", a.Slug))
		}
	}

	return nil
}

// Configure will configure a cron job for semver auto deployment schedules based on the application's configuration
func Configure(appID string) error {
	a, err := store.GetStore().GetApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get app")
	}

	if a.IsAirgap {
		return nil
	}

	if a.SemverAutoDeploy == "" || a.SemverAutoDeploy == apptypes.SemverAutoDeployDisabled {
		return nil
	}

	logger.Debug("configure semver auto deployments for app",
		zap.String("slug", a.Slug))

	mtx.Lock()
	defer mtx.Unlock()

	cronSpec := a.SemverAutoDeploySchedule

	if cronSpec == "" {
		Stop(a.ID)
		return nil
	}

	if cronSpec == "@default" {
		// if automatic deployments are enabled, then by default updates are automatically deployed as soon as they're available
		// if they meet the configured criteria, which happens as part of the automatic update check process
		Stop(a.ID)
		return nil
	}

	job, ok := jobs[a.ID]
	if ok {
		// job already exists, remove entries
		entries := job.Entries()
		for _, entry := range entries {
			job.Remove(entry.ID)
		}
	} else {
		// job does not exist, create a new one
		job = cron.New(cron.WithChain(
			cron.Recover(cron.DefaultLogger),
		))
	}

	jobAppID := a.ID
	jobAppSlug := a.Slug
	jobSemverAutoDeploy := a.SemverAutoDeploy

	_, err = job.AddFunc(cronSpec, func() {
		logger.Debug("processing semver auto deployments for app", zap.String("slug", jobAppSlug))

		opts := ExecuteOpts{
			AppID:            jobAppID,
			SemverAutoDeploy: jobSemverAutoDeploy,
		}
		if err := execute(opts); err != nil {
			logger.Error(errors.Wrapf(err, "failed to execute for app %s", jobAppSlug))
			return
		}
	})
	if err != nil {
		return errors.Wrap(err, "failed to add func")
	}

	job.Start()
	jobs[a.ID] = job

	return nil
}

// Stop will stop a running cron job (if exists) for a specific app
func Stop(appID string) {
	if jobs == nil {
		logger.Debug("no cron jobs found")
		return
	}
	if job, ok := jobs[appID]; ok {
		job.Stop()
	} else {
		logger.Debug("cron job not found for app", zap.String("appID", appID))
	}
}

type ExecuteOpts struct {
	AppID            string
	SemverAutoDeploy apptypes.SemverAutoDeploy
}

func execute(opts ExecuteOpts) error {
	a, err := store.GetStore().GetApp(opts.AppID)
	if err != nil {
		return errors.Wrap(err, "failed to get app")
	}

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to list downstreams for app")
	}
	if len(downstreams) == 0 {
		return errors.Errorf("no downstreams found for app %q", a.Slug)
	}
	d := downstreams[0]

	currentVersion, err := store.GetStore().GetCurrentVersion(a.ID, d.ClusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get current version")
	}
	if currentVersion == nil {
		return nil
	}

	versions, err := store.GetStore().GetAppVersions(a.ID, d.ClusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get pending versions")
	}

	sequence := findVersionToDeploy(opts, versions.PendingVersions, currentVersion.VersionLabel)
	if sequence == -1 {
		return nil
	}

	status, err := store.GetStore().GetStatusForVersion(a.ID, d.ClusterID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to get status for version")
	}

	if status == storetypes.VersionPendingConfig {
		logger.Infof("not deploying version %d because it's %s", sequence, status)
		return nil
	}

	if err := version.DeployVersion(a.ID, sequence); err != nil {
		return errors.Wrap(err, "failed to queue version for deployment")
	}

	return nil
}

type PendingVersion struct {
	Semver   semver.Version
	Sequence int64
}

type PendingVersions []PendingVersion

func (s PendingVersions) Len() int { return len(s) }

func (s PendingVersions) Less(i, j int) bool {
	return s[i].Semver.LT(s[j].Semver)
}

func (s PendingVersions) Swap(i, j int) {
	tmp := s[i]
	s[i] = s[j]
	s[j] = tmp
}

// findVersionToDeploy will return the sequence number for the version that satisfies the semver auto deploy criteria or -1 if no match was found
func findVersionToDeploy(opts ExecuteOpts, versions []*downstreamtypes.DownstreamVersion, currentVersionLabel string) int64 {
	currentSemver, err := semver.ParseTolerant(currentVersionLabel)
	if err != nil {
		return -1
	}

	pendingVersions := PendingVersions{}

	for _, v := range versions {
		switch opts.SemverAutoDeploy {
		case apptypes.SemverAutoDeployPatch:
			s, err := semver.ParseTolerant(v.VersionLabel)
			if err != nil {
				continue
			}
			if s.Major != currentSemver.Major || s.Minor != currentSemver.Minor {
				continue
			}
			pendingVersions = append(pendingVersions, PendingVersion{
				Semver:   s,
				Sequence: v.Sequence,
			})

		case apptypes.SemverAutoDeployMinorPatch:
			s, err := semver.ParseTolerant(v.VersionLabel)
			if err != nil {
				continue
			}
			if s.Major != currentSemver.Major {
				continue
			}
			pendingVersions = append(pendingVersions, PendingVersion{
				Semver:   s,
				Sequence: v.Sequence,
			})

		case apptypes.SemverAutoDeployMajorMinorPatch:
			s, err := semver.ParseTolerant(v.VersionLabel)
			if err != nil {
				continue
			}
			pendingVersions = append(pendingVersions, PendingVersion{
				Semver:   s,
				Sequence: v.Sequence,
			})
		}
	}

	if len(pendingVersions) == 0 {
		return -1
	}
	sort.Sort(sort.Reverse(pendingVersions))

	return pendingVersions[0].Sequence
}
