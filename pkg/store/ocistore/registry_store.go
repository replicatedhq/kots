package ocistore

import (
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
)

func (s *OCIStore) GetRegistryDetailsForApp(appID string) (registrytypes.RegistrySettings, error) {
	return registrytypes.RegistrySettings{}, ErrNotImplemented
}

func (s *OCIStore) UpdateRegistry(appID string, hostname string, username string, password string, namespace string, isReadOnly bool) error {
	return ErrNotImplemented
}

func (s *OCIStore) GetAppIDsFromRegistry(hostname string) ([]string, error) {
	return nil, ErrNotImplemented
}
