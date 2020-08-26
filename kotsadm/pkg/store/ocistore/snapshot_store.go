package ocistore

import (
	"time"
	"database/sql"

	snapshottypes "github.com/replicatedhq/kots/kotsadm/pkg/snapshot/types"
)

func (c OCIStore) ListPendingScheduledSnapshots(appID string) ([]snapshottypes.ScheduledSnapshot, error) {
	return nil, ErrNotImplemented
}

func (c OCIStore) UpdateScheduledSnapshot(_ *sql.Tx, ID string, backupName string) error {
	return ErrNotImplemented
}

func (c OCIStore) LockScheduledSnapshot(_ *sql.Tx, ID string) (bool, error) {
	return false, ErrNotImplemented
}

func (c OCIStore) DeletePendingScheduledSnapshots(appID string, _ *sql.Tx) error {
	return ErrNotImplemented
}

func (c OCIStore) CreateScheduledSnapshot(id string, appID string, timestamp time.Time, _ *sql.Tx) error {
	return ErrNotImplemented
}
