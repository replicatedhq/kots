package store

import (
	"github.com/replicatedhq/kots/kotsadm/pkg/store/s3pg"
)

var (
	hasStore    = false
	globalStore KOTSStore
)

var _ KOTSStore = (*s3pg.S3PGStore)(nil)

func GetStore() KOTSStore {
	if !hasStore {
		globalStore = s3pg.S3PGStore{}
	}

	return globalStore
}
