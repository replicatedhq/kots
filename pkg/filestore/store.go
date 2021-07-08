package filestore

import (
	"os"
)

var (
	hasStore    = false
	globalStore FileStore
)

func GetStore() FileStore {
	if !hasStore {
		globalStore = storeFromEnv()
		hasStore = true
	}

	return globalStore
}

func storeFromEnv() FileStore {
	if os.Getenv("S3_ENDPOINT") == "" {
		return &BlobStore{}
	}
	return &S3Store{}
}
