package ocistore

import (
	"time"
)

func (c OCIStore) DeletePendingScheduledSnapshots(appID string) error {
	return ErrNotImplemented
}

func (c OCIStore) CreateScheduledSnapshot(id string, appID string, timestamp time.Time) error {
	return ErrNotImplemented
}
