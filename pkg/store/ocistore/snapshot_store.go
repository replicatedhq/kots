package ocistore

import (
	"time"

	snapshottypes "github.com/replicatedhq/kots/pkg/kotsadmsnapshot/types"
)

func (c OCIStore) ListPendingScheduledSnapshots(appID string) ([]snapshottypes.ScheduledSnapshot, error) {
	return nil, ErrNotImplemented
}

func (c OCIStore) UpdateScheduledSnapshot(snapshotID string, backupName string) error {
	return ErrNotImplemented
}

func (c OCIStore) DeletePendingScheduledSnapshots(appID string) error {
	return ErrNotImplemented
}

func (c OCIStore) CreateScheduledSnapshot(snapshotID string, appID string, timestamp time.Time) error {
	return ErrNotImplemented
}

func (c OCIStore) ListPendingScheduledInstanceSnapshots(clusterID string) ([]snapshottypes.ScheduledInstanceSnapshot, error) {
	return nil, ErrNotImplemented
}

func (c OCIStore) UpdateScheduledInstanceSnapshot(snapshotID string, backupName string) error {
	return ErrNotImplemented
}

func (c OCIStore) DeletePendingScheduledInstanceSnapshots(clusterID string) error {
	return ErrNotImplemented
}

func (c OCIStore) CreateScheduledInstanceSnapshot(snapshotID string, clusterID string, timestamp time.Time) error {
	return ErrNotImplemented
}
