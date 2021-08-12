package ocistore

import (
	"time"

	appstatetypes "github.com/replicatedhq/kots/pkg/appstate/types"
)

func (s *OCIStore) GetAppStatus(appID string) (*appstatetypes.AppStatus, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) SetAppStatus(appID string, resourceStates appstatetypes.ResourceStates, updatedAt time.Time, sequence int64) error {
	return ErrNotImplemented
}
