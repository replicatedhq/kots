package ocistore

import (
	appstatustypes "github.com/replicatedhq/kots/kotsadm/pkg/appstatus/types"
)

func (s OCIStore) GetAppStatus(appID string) (*appstatustypes.AppStatus, error) {
	return nil, ErrNotImplemented
}
