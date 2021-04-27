package ocistore

import (
	"time"

	appstatustypes "github.com/replicatedhq/kots/pkg/api/appstatus/types"
)

func (s *OCIStore) GetAppStatus(appID string) (*appstatustypes.AppStatus, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) SetAppStatus(appID string, resourceStates []appstatustypes.ResourceState, updatedAt time.Time, sequence int64) error {
	return ErrNotImplemented
}
