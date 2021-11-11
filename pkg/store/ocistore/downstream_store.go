package ocistore

import (
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/store/types"
)

func (s *OCIStore) GetCurrentSequence(appID string, clusterID string) (int64, error) {
	return 0, ErrNotImplemented
}

func (s *OCIStore) GetCurrentParentSequence(appID string, clusterID string) (int64, error) {
	return 0, ErrNotImplemented
}

func (s *OCIStore) GetParentSequenceForSequence(appID string, clusterID string, sequence int64) (int64, error) {
	return 0, ErrNotImplemented
}

func (s *OCIStore) GetPreviouslyDeployedSequence(appID string, clusterID string) (int64, error) {
	return 0, ErrNotImplemented
}

func (s *OCIStore) SetDownstreamVersionReady(appID string, sequence int64) error {
	return ErrNotImplemented
}

func (s *OCIStore) SetDownstreamVersionPendingPreflight(appID string, sequence int64) error {
	return ErrNotImplemented
}

func (s *OCIStore) UpdateDownstreamVersionStatus(appID string, sequence int64, status string, statusInfo string) error {
	return ErrNotImplemented
}

func (s *OCIStore) GetDownstreamVersionStatus(appID string, sequence int64) (types.DownstreamVersionStatus, error) {
	return types.DownstreamVersionStatus(""), ErrNotImplemented
}

func (s *OCIStore) GetIgnoreRBACErrors(appID string, sequence int64) (bool, error) {
	return false, ErrNotImplemented
}

func (s *OCIStore) GetCurrentVersion(appID string, clusterID string) (*downstreamtypes.DownstreamVersion, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) GetStatusForVersion(appID string, clusterID string, sequence int64) (types.DownstreamVersionStatus, error) {
	return types.DownstreamVersionStatus(""), ErrNotImplemented
}

func (s *OCIStore) GetAppVersions(appID string, clusterID string) (*downstreamtypes.DownstreamVersions, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) GetDownstreamOutput(appID string, clusterID string, sequence int64) (*downstreamtypes.DownstreamOutput, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) IsDownstreamDeploySuccessful(appID string, clusterID string, sequence int64) (bool, error) {
	return false, ErrNotImplemented
}

func (s *OCIStore) UpdateDownstreamDeployStatus(appID string, clusterID string, sequence int64, isError bool, output downstreamtypes.DownstreamOutput) error {
	return ErrNotImplemented
}

func (s *OCIStore) DeleteDownstreamDeployStatus(appID string, clusterID string, sequence int64) error {
	return ErrNotImplemented
}
