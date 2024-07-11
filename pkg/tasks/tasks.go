package tasks

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/rqlite/gorqlite"
)

type TaskStatus struct {
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func StartTaskMonitor(taskID string, finishedChan <-chan error) {
	go func() {
		var finalError error
		defer func() {
			if finalError == nil {
				if err := ClearTaskStatus(taskID); err != nil {
					logger.Error(errors.Wrapf(err, "failed to clear %s task status", taskID))
				}
			} else {
				errMsg := finalError.Error()
				if cause, ok := errors.Cause(finalError).(util.ActionableError); ok {
					errMsg = cause.Error()
				}
				if err := SetTaskStatus(taskID, errMsg, "failed"); err != nil {
					logger.Error(errors.Wrapf(err, "failed to set error on %s task status", taskID))
				}
			}
		}()

		for {
			select {
			case <-time.After(time.Second * 2):
				if err := UpdateTaskStatusTimestamp(taskID); err != nil {
					logger.Error(err)
				}
			case err := <-finishedChan:
				finalError = err
				return
			}
		}
	}()
}

func StartTicker(taskID string, finishedChan <-chan struct{}) {
	go func() {
		for {
			select {
			case <-time.After(time.Second * 2):
				if err := UpdateTaskStatusTimestamp(taskID); err != nil {
					logger.Error(err)
				}
			case <-finishedChan:
				return
			}
		}
	}()
}

func SetTaskStatus(id string, message string, status string) error {
	db := persistence.MustGetDBSession()

	query := `
INSERT INTO api_task_status (id, updated_at, current_message, status)
VALUES (?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	updated_at = EXCLUDED.updated_at,
	current_message = EXCLUDED.current_message,
	status = EXCLUDED.status
`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{id, time.Now().Unix(), message, status},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}

func UpdateTaskStatusTimestamp(id string) error {
	db := persistence.MustGetDBSession()

	query := `UPDATE api_task_status SET updated_at = ? WHERE id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{time.Now().Unix(), id},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}

func ClearTaskStatus(id string) error {
	db := persistence.MustGetDBSession()

	query := `DELETE FROM api_task_status WHERE id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{id},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}

func GetTaskStatus(id string) (string, string, error) {
	db := persistence.MustGetDBSession()

	// only return the status if it was updated in the last minute
	query := `SELECT status, current_message from api_task_status WHERE id = ? AND strftime('%s', 'now') - updated_at < 60`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{id},
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to query app: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return "", "", nil
	}

	var status gorqlite.NullString
	var message gorqlite.NullString
	if err := rows.Scan(&status, &message); err != nil {
		return "", "", errors.Wrap(err, "failed to scan")
	}

	return status.String, message.String, nil
}
