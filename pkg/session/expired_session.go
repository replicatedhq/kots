package session

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/robfig/cron/v3"
)

const (
	// purgeExpiredSessionsCronSpec - daily cron spec for the session purge job
	purgeExpiredSessionsCronSpec = "0 0 * * *"
)

// StartSessionPurgeCronJob - start the session purge cron job which deletes expired sessions periodically according to the cron spec above
func StartSessionPurgeCronJob() error {
	logger.Debug("starting sessions purge cron job")

	cronJob := cron.New(cron.WithChain(
		cron.Recover(cron.DefaultLogger),
	))

	_, err := cronJob.AddFunc(purgeExpiredSessionsCronSpec, func() {
		logger.Debug("running expired sessions purge job")
		err := deleteExpiredSessions()
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to delete expired sessions"))
		}
	})
	if err != nil {
		return errors.Wrap(err, "failed to add cron job")
	}
	cronJob.Start()
	return nil
}

// deleteExpiredSessions - delete all expired sessions
func deleteExpiredSessions() error {
	store := store.GetStore()
	err := store.DeleteExpiredSessions()
	if err != nil {
		return errors.Wrap(err, "failed to delete expired sessions")
	}
	return nil
}
