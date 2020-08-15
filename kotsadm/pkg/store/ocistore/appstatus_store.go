package ocistore

import (
	"github.com/pkg/errors"
	appstatustypes "github.com/replicatedhq/kots/kotsadm/pkg/appstatus/types"
)

func (s OCIStore) GetAppStatus(appID string) (*appstatustypes.AppStatus, error) {
	return nil, errors.New("not implemented")
}
