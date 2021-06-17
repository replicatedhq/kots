package filestore

import (
	"github.com/replicatedhq/kots/pkg/filestore/blobstore"
	"github.com/replicatedhq/kots/pkg/filestore/s3store"
)

var (
	hasStore    = false
	globalStore FileStore
)

var _ FileStore = (*s3store.S3Store)(nil)
var _ FileStore = (*blobstore.BlobStore)(nil)

func GetStore() FileStore {
	if !hasStore {
		globalStore = storeFromEnv()
		hasStore = true
	}

	return globalStore
}

func storeFromEnv() FileStore {
	return &blobstore.BlobStore{}
}
