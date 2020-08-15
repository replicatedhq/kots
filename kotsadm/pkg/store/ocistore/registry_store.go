package ocistore

import (
	"github.com/pkg/errors"
	registrytypes "github.com/replicatedhq/kots/kotsadm/pkg/registry/types"
)

func (s OCIStore) GetRegistryDetailsForApp(appID string) (*registrytypes.RegistrySettings, error) {
	return nil, errors.New("not implemented")
}

func (s OCIStore) UpdateRegistry(appID string, hostname string, username string, password string, namespace string) error {
	return errors.New("not implemented")
}
