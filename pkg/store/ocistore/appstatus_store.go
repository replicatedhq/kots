package ocistore

import (
	appstatustypes "github.com/replicatedhq/kots/pkg/api/appstatus/types"
)

func (s OCIStore) GetAppStatus(appID string) (*appstatustypes.AppStatus, error) {
	return nil, ErrNotImplemented
}

func (s OCIStore) SetAppStatus(appID string) error {
	return ErrNotImplemented
}
