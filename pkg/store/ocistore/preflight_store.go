package ocistore

import (
	preflighttypes "github.com/replicatedhq/kots/pkg/preflight/types"
)

func (s OCIStore) SetPreflightResults(appID string, sequence int64, results []byte) error {
	return ErrNotImplemented
}

func (s OCIStore) GetPreflightResults(appID string, sequence int64) (*preflighttypes.PreflightResult, error) {
	return nil, ErrNotImplemented
}

func (s OCIStore) GetLatestPreflightResultsForSequenceZero() (*preflighttypes.PreflightResult, error) {
	return nil, ErrNotImplemented
}

func (s OCIStore) ResetPreflightResults(appID string, sequence int64) error {
	return ErrNotImplemented
}

func (s OCIStore) SetIgnorePreflightPermissionErrors(appID string, sequence int64) error {
	return ErrNotImplemented
}
