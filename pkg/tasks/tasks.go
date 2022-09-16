package tasks

import (
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
)

func StartUpdateTaskMonitor(taskID string, finishedChan <-chan error) {
	go func() {
		var finalError error
		defer func() {
			if finalError == nil {
				if err := store.GetStore().ClearTaskStatus(taskID); err != nil {
					logger.Error(errors.Wrap(err, "failed to clear update-download task status"))
				}
			} else {
				errMsg := finalError.Error()
				if cause, ok := errors.Cause(finalError).(util.ActionableError); ok {
					errMsg = cause.Error()
				}
				if err := store.GetStore().SetTaskStatus(taskID, errMsg, "failed"); err != nil {
					logger.Error(errors.Wrap(err, "failed to set error on update-download task status"))
				}
			}
		}()

		for {
			select {
			case <-time.After(time.Second):
				if err := store.GetStore().UpdateTaskStatusTimestamp(taskID); err != nil {
					logger.Error(err)
				}
			case err := <-finishedChan:
				finalError = err
				return
			}
		}
	}()
}
