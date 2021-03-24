package ocistore

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/pkg/errors"
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

func (s *OCIStore) SetTaskStatus(id string, message string, status string) error {
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

func (s *OCIStore) UpdateTaskStatusTimestamp(id string) error {
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

func (s *OCIStore) ClearTaskStatus(id string) error {
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

func (s *OCIStore) GetTaskStatus(id string) (string, string, error) {
	taskStatusLock.Lock()
	defer taskStatusLock.Unlock()

	cached := s.cachedTaskStatus[id]
	if cached != nil && time.Now().Before(cached.expirationTime) {
		return cached.taskStatus.Status, cached.taskStatus.Message, nil
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

	if cached == nil {
		cached = &cachedTaskStatus{}
		s.cachedTaskStatus[id] = cached
	}
	cached.taskStatus = ts
	cached.expirationTime = time.Now().Add(taskCacheTTL)

	return ts.Status, ts.Message, nil
}
