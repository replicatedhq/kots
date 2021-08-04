package ocistore

func (s *OCIStore) GetSharedPasswordBcrypt() ([]byte, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) FlagInvalidPassword() error {
	return ErrNotFound
}

func (s *OCIStore) FlagSuccessfulLogin() error {
	return ErrNotImplemented
}
