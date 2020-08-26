package ocistore

import (
	"encoding/json"

	"github.com/pkg/errors"
)

const (
	TaskStatusConfigMapName = `kotsadm-tasks`
)

type taskStatus struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

func (s OCIStore) SetTaskStatus(id string, message string, status string) error {
	return ErrNotImplemented
}

func (s OCIStore) UpdateTaskStatusTimestamp(id string) error {
	return ErrNotImplemented
}

func (s OCIStore) ClearTaskStatus(id string) error {
	return ErrNotImplemented
}

func (s OCIStore) GetTaskStatus(id string) (string, string, error) {
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
