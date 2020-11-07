package ocistore

import (
	"context"
	"database/sql"
	"time"

	"github.com/ocidb/ocidb/pkg/ocidb"
	"github.com/pkg/errors"
)

type taskStatus struct {
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (s OCIStore) SetTaskStatus(id string, message string, status string) error {
	query := `insert into api_task_status (id, updated_at, current_message, status) values ($1, $2, $3, $4)
on conflict(id) do update set current_message = EXCLUDED.current_message, status = EXCLUDED.status`

	_, err := s.connection.DB.Exec(query, id, time.Now(), message, status)
	if err != nil {
		return errors.Wrap(err, "failed to set task status")
	}
	if err := ocidb.Commit(context.TODO(), s.connection); err != nil {
		return errors.Wrap(err, "failed to commit")
	}

	return nil
}

func (s OCIStore) UpdateTaskStatusTimestamp(id string) error {
	query := `update api_task_status set updated_at = $1 where id = $2`

	_, err := s.connection.DB.Exec(query, time.Now(), id)
	if err != nil {
		return errors.Wrap(err, "failed to update task status")
	}
	if err := ocidb.Commit(context.TODO(), s.connection); err != nil {
		return errors.Wrap(err, "failed to commit")
	}

	return nil
}

func (s OCIStore) ClearTaskStatus(id string) error {
	query := `delete from api_task_status where id = $1`

	_, err := s.connection.DB.Exec(query, id)
	if err != nil {
		return errors.Wrap(err, "failed to clear task status")
	}
	if err := ocidb.Commit(context.TODO(), s.connection); err != nil {
		return errors.Wrap(err, "failed to commit")
	}

	return nil
}

func (s OCIStore) GetTaskStatus(id string) (string, string, error) {
	query := `select status, current_message from api_task_status where id = $1 AND updated_at > ($2::timestamp - '10 seconds'::interval)`
	row := s.connection.DB.QueryRow(query, id, time.Now())

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
