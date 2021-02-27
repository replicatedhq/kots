package kotsstore

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/persistence"
)

const (
	TaskStatusConfigMapName = `kotsadm-tasks`
)

type taskStatus struct {
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (s KOTSStore) migrationTasksFromPostgres() error {
	db := persistence.MustGetPGSession()

	query := `select updated_at, current_message, status from api_task_status`
	rows, err := db.Query(query)
	if err != nil {
		return errors.Wrap(err, "failed to select tasks for migration")
	}

	configmap, err := s.getConfigmap(TaskStatusConfigMapName)
	if err != nil {
		return errors.Wrap(err, "failed to get task status configmap")
	}

	if configmap.Data == nil {
		configmap.Data = map[string]string{}
	}

	for rows.Next() {
		var id string
		var status sql.NullString
		var message sql.NullString

		ts := taskStatus{}
		if err := rows.Scan(&id, &ts.UpdatedAt, &message, &status); err != nil {
			return errors.Wrap(err, "failed to scan task status")
		}

		if status.Valid {
			ts.Status = status.String
		}
		if message.Valid {
			ts.Message = message.String
		}

		b, err := json.Marshal(ts)
		if err != nil {
			return errors.Wrap(err, "failed to marshal task status")
		}

		configmap.Data[id] = string(b)
	}

	if err := s.updateConfigmap(configmap); err != nil {
		return errors.Wrap(err, "failed to update task status configmap")
	}

	query = `delete from api_task_status`
	if _, err := db.Exec(query); err != nil {
		return errors.Wrap(err, "failed to delete tasks from postgres")
	}

	return nil
}

func (s KOTSStore) SetTaskStatus(id string, message string, status string) error {
	configmap, err := s.getConfigmap(TaskStatusConfigMapName)
	if err != nil {
		return errors.Wrap(err, "failed to get task status configmap")
	}

	if configmap.Data == nil {
		configmap.Data = map[string]string{}
	}

	ts := taskStatus{}
	existingTsData, ok := configmap.Data[id]
	if ok {
		if err := json.Unmarshal([]byte(existingTsData), &ts); err != nil {
			return errors.Wrap(err, "failed to unmarshal task status")
		}
	}

	ts.Message = message
	ts.Status = status
	ts.UpdatedAt = time.Now()

	b, err := json.Marshal(ts)
	if err != nil {
		return errors.Wrap(err, "failed to marshal task status")
	}

	configmap.Data[id] = string(b)

	if err := s.updateConfigmap(configmap); err != nil {
		return errors.Wrap(err, "failed to update task status configmap")
	}

	return nil
}

func (s KOTSStore) UpdateTaskStatusTimestamp(id string) error {
	configmap, err := s.getConfigmap(TaskStatusConfigMapName)
	if err != nil {
		return errors.Wrap(err, "failed to get task status configmap")
	}

	if configmap.Data == nil {
		configmap.Data = map[string]string{}
	}

	data, ok := configmap.Data[id]
	if !ok {
		return nil // copied from s3pgstore
	}

	ts := taskStatus{}
	if err := json.Unmarshal([]byte(data), &ts); err != nil {
		return errors.Wrap(err, "failed to unmarshal task status")
	}

	ts.UpdatedAt = time.Now()

	b, err := json.Marshal(ts)
	if err != nil {
		return errors.Wrap(err, "failed to marshal task status")
	}

	configmap.Data[id] = string(b)

	if err := s.updateConfigmap(configmap); err != nil {
		return errors.Wrap(err, "failed to update task status configmap")
	}

	return nil
}

func (s KOTSStore) ClearTaskStatus(id string) error {
	configmap, err := s.getConfigmap(TaskStatusConfigMapName)
	if err != nil {
		return errors.Wrap(err, "failed to get task status configmap")
	}

	if configmap.Data == nil {
		configmap.Data = map[string]string{}
	}

	_, ok := configmap.Data[id]
	if !ok {
		return nil // copied from s3pgstore
	}

	delete(configmap.Data, id)

	if err := s.updateConfigmap(configmap); err != nil {
		return errors.Wrap(err, "failed to update task status configmap")
	}

	return nil
}

func (s KOTSStore) GetTaskStatus(id string) (string, string, error) {
	configmap, err := s.getConfigmap(TaskStatusConfigMapName)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get task status configmap")
	}

	if configmap.Data == nil {
		configmap.Data = map[string]string{}
	}

	marshalled, ok := configmap.Data[id]
	if !ok {
		return "", "", nil
	}

	ts := taskStatus{}
	if err := json.Unmarshal([]byte(marshalled), &ts); err != nil {
		return "", "", errors.Wrap(err, "error unmarshalling task status")
	}

	return ts.Status, ts.Message, nil
}
