package app

import (
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/persistence"
)

func SetTaskStatus(id string, message string, status string) error {
	db := persistence.MustGetPGSession()
	query := `insert into api_task_status (id, updated_at, current_message, status) values ($1, $2, $3, $4)
on conflict(id) do update set current_message = EXCLUDED.current_message, status = EXCLUDED.status`

	_, err := db.Exec(query, id, time.Now(), message, status)
	if err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	return nil
}

func UpdateTaskStatusTimestamp(id string) error {
	db := persistence.MustGetPGSession()
	query := `update api_task_status set updated_at = $1 where id = $2`

	_, err := db.Exec(query, time.Now(), id)
	if err != nil {
		return errors.Wrap(err, "failed to update task status")
	}

	return nil
}

func ClearTaskStatus(id string) error {
	db := persistence.MustGetPGSession()
	query := `delete from api_task_status where id = $1`

	_, err := db.Exec(query, id)
	if err != nil {
		return errors.Wrap(err, "failed to clear task status")
	}

	return nil
}

func GetTaskStatus(id string) (string, error) {
	db := persistence.MustGetPGSession()
	query := `select status from api_task_status where id = $1`

	row := db.QueryRow(query, id)
	status := ""
	if err := row.Scan(&status); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}

		return "", errors.Wrap(err, "failed to scan task status")
	}

	return status, nil
}
