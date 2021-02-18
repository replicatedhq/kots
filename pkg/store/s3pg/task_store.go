package s3pg

import (
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/persistence"
)

func (s S3PGStore) SetTaskStatus(id string, message string, status string) error {
	db := persistence.MustGetPGSession()
	query := `insert into api_task_status (id, updated_at, current_message, status) values ($1, $2, $3, $4)
on conflict(id) do update set current_message = EXCLUDED.current_message, status = EXCLUDED.status`

	_, err := db.Exec(query, id, time.Now(), message, status)
	if err != nil {
		return errors.Wrap(err, "failed to set task status")
	}

	return nil
}

func (s S3PGStore) UpdateTaskStatusTimestamp(id string) error {
	db := persistence.MustGetPGSession()
	query := `update api_task_status set updated_at = $1 where id = $2`

	_, err := db.Exec(query, time.Now(), id)
	if err != nil {
		return errors.Wrap(err, "failed to update task status")
	}

	return nil
}

func (s S3PGStore) ClearTaskStatus(id string) error {
	db := persistence.MustGetPGSession()
	query := `delete from api_task_status where id = $1`

	_, err := db.Exec(query, id)
	if err != nil {
		return errors.Wrap(err, "failed to clear task status")
	}

	return nil
}

func (s S3PGStore) GetTaskStatus(id string) (string, string, error) {
	db := persistence.MustGetPGSession()
	query := `select status, current_message from api_task_status where id = $1 AND updated_at > ($2::timestamp - '10 seconds'::interval)`
	row := db.QueryRow(query, id, time.Now())

	var status sql.NullString
	var message sql.NullString

	if err := row.Scan(&status, &message); err != nil {
		if err == sql.ErrNoRows {
			return "", "", nil
		}

		return "", "", errors.Wrap(err, "failed to scan task status")
	}

	return status.String, message.String, nil
}
