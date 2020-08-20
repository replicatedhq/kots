package ocistore

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
	return "", "", ErrNotImplemented
}
