package ocistore

import (
	preflighttypes "github.com/replicatedhq/kots/pkg/preflight/types"
)

func (s *OCIStore) SetPreflightProgress(appID string, sequence float64, progress string) error {
	return ErrNotImplemented
}

func (s *OCIStore) GetPreflightProgress(appID string, sequence float64) (string, error) {
	return "", ErrNotImplemented
}

func (s *OCIStore) SetPreflightResults(appID string, sequence float64, results []byte) error {
	return ErrNotImplemented
}

func (s *OCIStore) GetPreflightResults(appID string, sequence float64) (*preflighttypes.PreflightResult, error) {
	return nil, ErrNotImplemented
}

func (s *OCIStore) ResetPreflightResults(appID string, sequence float64) error {
	return ErrNotImplemented
}

func (s *OCIStore) SetIgnorePreflightPermissionErrors(appID string, sequence float64) error {
	return ErrNotImplemented
}
