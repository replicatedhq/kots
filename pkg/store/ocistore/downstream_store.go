package ocistore

import (
	"github.com/replicatedhq/kots/pkg/api/downstream/types"
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

func (s *OCIStore) GetDownstreamVersionStatus(appID string, sequence int64) (string, error) {
	return "", ErrNotImplemented
}

func (s *OCIStore) GetIgnoreRBACErrors(appID string, sequence int64) (bool, error) {
	return false, ErrNotImplemented
}

func (s *OCIStore) GetCurrentVersion(appID string, clusterID string) (*types.DownstreamVersion, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) GetStatusForVersion(appID string, clusterID string, sequence int64) (string, error) {
	return "", ErrNotImplemented
}

func (s *OCIStore) GetPendingVersions(appID string, clusterID string) ([]types.DownstreamVersion, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) GetPastVersions(appID string, clusterID string) ([]types.DownstreamVersion, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) GetDownstreamOutput(appID string, clusterID string, sequence int64) (*types.DownstreamOutput, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) IsDownstreamDeploySuccessful(appID string, clusterID string, sequence int64) (bool, error) {
	return false, ErrNotImplemented
}

func (s *OCIStore) UpdateDownstreamDeployStatus(appID string, clusterID string, sequence int64, isError bool, output types.DownstreamOutput) error {
	return ErrNotImplemented
}

func (s *OCIStore) DeleteDownstreamDeployStatus(appID string, clusterID string, sequence int64) error {
	return ErrNotImplemented
}
