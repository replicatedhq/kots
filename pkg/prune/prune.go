// Package prune implements an operator-controlled cleanup policy for old KOTS
// support bundle archives and application version archives.
//
// It is an undocumented escape hatch for operators under storage pressure: it is
// disabled by default and is configured by editing the kotsadm-confg ConfigMap
// directly (see kotsutil.InstallationParams and the prune-* keys). When enabled,
// one cleanup pass runs immediately at startup to address any existing backlog and
// then recurs daily.
package prune

import (
	"fmt"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/filestore"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	storepkg "github.com/replicatedhq/kots/pkg/store"
	cron "github.com/robfig/cron/v3"
)

// deleteDelay is the pause between individual archive/metadata deletions, to avoid
// a thundering-herd of delete requests against the object store during a pass.
const deleteDelay = 250 * time.Millisecond

type config struct {
	supportBundleCount int
	appVersionCount    int
	deleteDelay        time.Duration
}

// Start reads the prune configuration from the kotsadm-confg ConfigMap. If prune is
// disabled, it is a no-op. If enabled, it runs one cleanup pass immediately (in the
// background, off the startup path) and schedules a daily recurring pass.
func Start() error {
	params, err := kotsutil.GetInstallationParams(kotsadmtypes.KotsadmConfigMap)
	if err != nil {
		return errors.Wrap(err, "failed to get installation params")
	}

	if !params.PruneEnabled {
		return nil
	}

	cfg := config{
		supportBundleCount: params.PruneSupportBundleCount,
		appVersionCount:    params.PruneAppVersionCount,
		deleteDelay:        deleteDelay,
	}

	logger.Infof("prune enabled: retaining %d support bundles and %d app versions per app", cfg.supportBundleCount, cfg.appVersionCount)

	// run an immediate pass to address any existing backlog, off the startup path.
	// recover explicitly here: store calls reach persistence.MustGetDBSession, which
	// panics on an rqlite error, and an unrecovered panic in this goroutine would take
	// down the whole apiserver. The cron path below gets the same protection from
	// cron.Recover.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("prune: recovered from panic in initial pass: %v", r)
			}
		}()
		run(storepkg.GetStore(), filestore.GetStore(), cfg)
	}()

	job := cron.New(cron.WithChain(cron.Recover(cron.DefaultLogger)))
	if _, err := job.AddFunc("@daily", func() {
		run(storepkg.GetStore(), filestore.GetStore(), cfg)
	}); err != nil {
		return errors.Wrap(err, "failed to schedule prune job")
	}
	job.Start()

	return nil
}

// run performs a single prune pass across all installed apps. Errors are logged and
// skipped so that one bad app or object does not abort the whole pass.
func run(s storepkg.Store, fs filestore.FileStore, cfg config) {
	logger.Debug("starting prune pass")

	apps, err := s.ListInstalledApps()
	if err != nil {
		logger.Error(errors.Wrap(err, "prune: failed to list installed apps"))
		return
	}

	for _, app := range apps {
		pruneAppVersions(s, fs, app.ID, cfg)
		pruneSupportBundles(s, app.ID, cfg)
	}

	logger.Debug("finished prune pass")
}

// pruneAppVersions deletes old app version archives (metadata + object) for a single
// app. It never deletes the currently-deployed version nor any version newer than it
// (which may be pending deployment); of the remaining older versions it keeps the
// newest cfg.appVersionCount and deletes the rest.
//
// "Older" and "newer" are ordered by app_version sequence (assigned at creation/pull
// time), not by semver. A version created before the deployed one but carrying a higher
// semver (e.g. a backport) is treated as older and is eligible for pruning; the
// currently-deployed version is always protected regardless of ordering.
func pruneAppVersions(s storepkg.Store, fs filestore.FileStore, appID string, cfg config) {
	dvs, err := s.FindDownstreamVersions(appID, false)
	if err != nil {
		logger.Error(errors.Wrapf(err, "prune: failed to list versions for app %s", appID))
		return
	}
	if dvs == nil || dvs.CurrentVersion == nil {
		// nothing deployed yet; don't prune
		return
	}

	currentParentSequence := dvs.CurrentVersion.ParentSequence

	// collect distinct parent sequences strictly older than the deployed version.
	// Protecting everything >= the deployed sequence keeps the deployed version and any
	// newer, not-yet-deployed versions that may still be needed.
	seen := map[int64]bool{}
	prunable := []int64{}
	for _, v := range dvs.AllVersions {
		if v.ParentSequence >= currentParentSequence {
			continue
		}
		if seen[v.ParentSequence] {
			continue
		}
		seen[v.ParentSequence] = true
		prunable = append(prunable, v.ParentSequence)
	}

	// keep the newest cfg.appVersionCount of the prunable (older) versions
	sort.Slice(prunable, func(i, j int) bool { return prunable[i] > prunable[j] })
	if len(prunable) <= cfg.appVersionCount {
		return
	}
	toDelete := prunable[cfg.appVersionCount:]

	for _, sequence := range toDelete {
		logger.Infof("prune: deleting app version for app %s sequence %d", appID, sequence)

		// delete metadata first, then the archive: if the archive delete fails the
		// orphaned object is reclaimed on a later pass, whereas the reverse would leave
		// a dangling record whose archive fetch would 404.
		if err := s.DeleteAppVersion(appID, sequence); err != nil {
			logger.Error(errors.Wrapf(err, "prune: failed to delete app version metadata for app %s sequence %d", appID, sequence))
			continue
		}

		archivePath := fmt.Sprintf("%s/%d.tar.gz", appID, sequence)
		if err := fs.DeleteArchive(archivePath); err != nil {
			logger.Error(errors.Wrapf(err, "prune: failed to delete app version archive %s", archivePath))
		}

		time.Sleep(cfg.deleteDelay)
	}
}

// pruneSupportBundles deletes old support bundles (Secret + archive) for a single app,
// keeping the newest cfg.supportBundleCount.
func pruneSupportBundles(s storepkg.Store, appID string, cfg config) {
	bundles, err := s.ListSupportBundles(appID)
	if err != nil {
		logger.Error(errors.Wrapf(err, "prune: failed to list support bundles for app %s", appID))
		return
	}

	// ListSupportBundles returns newest-first, so anything past the retention count is old.
	if len(bundles) <= cfg.supportBundleCount {
		return
	}

	for _, bundle := range bundles[cfg.supportBundleCount:] {
		logger.Infof("prune: deleting support bundle %s for app %s", bundle.ID, appID)

		if err := s.DeleteSupportBundle(bundle.ID, appID); err != nil {
			logger.Error(errors.Wrapf(err, "prune: failed to delete support bundle %s", bundle.ID))
			continue
		}
		time.Sleep(cfg.deleteDelay)
	}
}
