package ocistore

import (
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
)

func (s *OCIStore) GetRegistryDetailsForApp(appID string) (*registrytypes.RegistrySettings, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) UpdateRegistry(appID string, hostname string, username string, password string, namespace string) error {
	return ErrNotImplemented
}
