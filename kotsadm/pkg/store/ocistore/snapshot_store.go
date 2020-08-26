package ocistore

import (
	"time"

	snapshottypes "github.com/replicatedhq/kots/kotsadm/pkg/snapshot/types"
)

func (c OCIStore) ListPendingScheduledSnapshots(appID string) ([]snapshottypes.ScheduledSnapshot, error) {
	return nil, ErrNotImplemented
}

func (c OCIStore) UpdateScheduledSnapshot(ID string, backupName string) error {
	return ErrNotImplemented
}

func (c OCIStore) DeletePendingScheduledSnapshots(appID string) error {
	return ErrNotImplemented
}

func (c OCIStore) CreateScheduledSnapshot(id string, appID string, timestamp time.Time) error {
	return ErrNotImplemented
}
