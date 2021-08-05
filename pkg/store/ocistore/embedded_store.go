package ocistore

func (s *OCIStore) GetEmbeddedClusterAuthToken() (string, error) {
	return "", ErrNotImplemented
}

func (s *OCIStore) SetEmbeddedClusterAuthToken(token string) error {
	return ErrNotImplemented
}
