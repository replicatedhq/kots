package ocistore

import "github.com/pkg/errors"

func (s OCIStore) SetTaskStatus(id string, message string, status string) error {
	return errors.New("not implemented")
}

func (s OCIStore) UpdateTaskStatusTimestamp(id string) error {
	return errors.New("not implemented")
}

func (s OCIStore) ClearTaskStatus(id string) error {
	return errors.New("not implemented")
}

func (s OCIStore) GetTaskStatus(id string) (string, string, error) {
	return "", "", errors.New("not implemented")
}
