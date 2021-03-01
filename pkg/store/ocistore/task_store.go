package ocistore

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

const (
	TaskStatusConfigMapName = `kotsadm-tasks`
)

type taskStatus struct {
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (s OCIStore) SetTaskStatus(id string, message string, status string) error {
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

func (s OCIStore) UpdateTaskStatusTimestamp(id string) error {
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

func (s OCIStore) ClearTaskStatus(id string) error {
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

func (s OCIStore) GetTaskStatus(id string) (string, string, error) {
	configmap, err := s.getConfigmap(TaskStatusConfigMapName)
	if err != nil {
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

	return ts.Status, ts.Message, nil
}
