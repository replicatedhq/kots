package kotsstore

import (
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/persistence"
)

const (
	TaskStatusConfigMapName = `kotsadm-tasks`

	taskCacheTTL = 1 * time.Minute
)

var (
	taskStatusLock = sync.Mutex{}
)

type taskStatus struct {
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (s *KOTSStore) migrationTasksFromPostgres() error {
	db := persistence.MustGetDBSession()

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

func (s *KOTSStore) SetTaskStatus(id string, message string, status string) error {
	taskStatusLock.Lock()
	defer taskStatusLock.Unlock()

	cached := s.cachedTaskStatus[id]
	if cached == nil {
		cached = &cachedTaskStatus{}
		s.cachedTaskStatus[id] = cached
	}
	cached.taskStatus.Message = message
	cached.taskStatus.Status = status
	cached.taskStatus.UpdatedAt = time.Now()
	cached.expirationTime = time.Now().Add(taskCacheTTL)

	configmap, err := s.getConfigmap(TaskStatusConfigMapName)
	if err != nil {
		if canIgnoreEtcdError(err) {
			return nil
		}
		return errors.Wrap(err, "failed to get task status configmap")
	}

	if configmap.Data == nil {
		configmap.Data = map[string]string{}
	}

	b, err := json.Marshal(cached.taskStatus)
	if err != nil {
		return errors.Wrap(err, "failed to marshal task status")
	}

	configmap.Data[id] = string(b)

	if err := s.updateConfigmap(configmap); err != nil {
		if canIgnoreEtcdError(err) {
			return nil
		}
		return errors.Wrap(err, "failed to update task status configmap")
	}

	return nil
}

func (s *KOTSStore) UpdateTaskStatusTimestamp(id string) error {
	taskStatusLock.Lock()
	defer taskStatusLock.Unlock()

	cached := s.cachedTaskStatus[id]
	if cached != nil {
		cached.taskStatus.UpdatedAt = time.Now()
		cached.expirationTime = time.Now().Add(taskCacheTTL)
	}

	configmap, err := s.getConfigmap(TaskStatusConfigMapName)
	if err != nil {
		if canIgnoreEtcdError(err) && cached != nil {
			return nil
		}
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
		if canIgnoreEtcdError(err) && cached != nil {
			return nil
		}
		return errors.Wrap(err, "failed to update task status configmap")
	}

	return nil
}

func (s *KOTSStore) ClearTaskStatus(id string) error {
	taskStatusLock.Lock()
	defer taskStatusLock.Unlock()

	defer delete(s.cachedTaskStatus, id)

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

func (s *KOTSStore) GetTaskStatus(id string) (string, string, error) {
	taskStatusLock.Lock()
	defer taskStatusLock.Unlock()

	cached := s.cachedTaskStatus[id]
	if cached != nil && time.Now().Before(cached.expirationTime) {
		return cached.taskStatus.Status, cached.taskStatus.Message, nil
	}

	if cached == nil {
		cached = &cachedTaskStatus{
			expirationTime: time.Now().Add(taskCacheTTL),
		}
		s.cachedTaskStatus[id] = cached
	}

	configmap, err := s.getConfigmap(TaskStatusConfigMapName)
	if err != nil {
		if canIgnoreEtcdError(err) && cached != nil {
			return cached.taskStatus.Status, cached.taskStatus.Message, nil
		}
		return "", "", errors.Wrap(err, "failed to get task status configmap")
	}

	if configmap.Data == nil {
		return "", "", nil
	}

	marshalled, ok := configmap.Data[id]
	if !ok {
		return "", "", nil
	}

	ts := taskStatus{}
	if err := json.Unmarshal([]byte(marshalled), &ts); err != nil {
		return "", "", errors.Wrap(err, "error unmarshalling task status")
	}

	if ts.UpdatedAt.Before(time.Now().Add(-10 * time.Second)) {
		return "", "", nil
	}

	cached.taskStatus = ts

	return ts.Status, ts.Message, nil
}
