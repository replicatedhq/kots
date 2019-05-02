package reconciler

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship-operator/pkg/ship"
)

type actions struct {
	addSecretMeta    bool
	removeSecretMeta bool
	deleteWatchJob   bool
	deleteUpdateJob  bool
	createWatchJob   bool
	createUpdateJob  bool
}

func (r *Reconciler) getActions() actions {
	debug := level.Debug(log.With(r.logger, "method", "reconcilers"))
	actions := actions{}

	if r.instance == nil {
		// The instance has been deleted. Remove all metadata added to the
		// Secret. The Jobs have owner references to the instance so they will
		// be garbage collected automatically.
		if r.secret != nil && ship.HasSecretMeta(r.secret, r.instanceName) {
			// TODO this should be a finalizer
			actions.removeSecretMeta = true
		}
		return actions // everything below assumes the instance has not been deleted
	}

	if r.secret != nil && !ship.HasSecretMeta(r.secret, r.instanceName) {
		debug.Log("reconciler", "addSecretMeta", "reason", "secret specified by ShipWatch instance exists but lacks labels or annotations")
		actions.addSecretMeta = true
	}

	if jobIsRunning(r.updateJob) && r.watchJob != nil {
		debug.Log("reconciler", "deleteWatchJob", "reason", "update Job is running")
		actions.deleteWatchJob = true
	} else if jobIsRunning(r.watchJob) && r.updateJob != nil {
		debug.Log("reconciler", "deleteUpdateJob", "reason", "watch Job is running")
		actions.deleteUpdateJob = true
	} else if jobIsRunning(r.watchJob) {
		desiredWatchJob := r.generator.WatchJob(r.stateSecretSHA())
		if r.shouldUpdateJob(r.watchJob, desiredWatchJob) {
			debug.Log("reconciler", "updateWatchJob", "reason", "running watch Job does not match desired config")
			actions.deleteWatchJob = true
			actions.createWatchJob = true
		}
	}

	if jobIsComplete(r.watchJob) {
		if r.updateJob == nil {
			debug.Log("reconciler", "createUpdateJob", "reason", "watch Job has completed")
			actions.createUpdateJob = true
		}
	}
	if jobIsComplete(r.updateJob) {
		if r.watchJob == nil {
			debug.Log("reconciler", "createWatchJob", "reason", "update Job has completed")
			actions.createWatchJob = true
		}
	}
	if jobIsComplete(r.updateJob) && jobIsComplete(r.watchJob) {
		debug.Log("reconciler", "deleteWatchJob", "reason", "update and watch Jobs have completed")
		actions.deleteWatchJob = true
		// after the watch job is deleted, a new one will be created by the 'updateJob complete' code above
		// after that job is running, the update job will be deleted by the 'updateJob not running && not nil && watchJob running' code above
		// this ensures that we never accidentally rerun an update (from deleting the update job before the watch job)
	}

	if r.updateJob == nil && r.watchJob == nil {
		// TODO should the update job ever be created instead?
		debug.Log("reconciler", "createWatchJob", "reason", "no Jobs for instance")
		actions.createWatchJob = true
	}

	return actions
}
