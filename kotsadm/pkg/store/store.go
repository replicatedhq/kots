package store

import (
	"fmt"
	"net/url"
	"os"

	"github.com/replicatedhq/kots/kotsadm/pkg/store/ocistore"
	"github.com/replicatedhq/kots/kotsadm/pkg/store/s3pg"
)

var (
	hasStore    = false
	globalStore KOTSStore
)

var _ KOTSStore = (*s3pg.S3PGStore)(nil)
var _ KOTSStore = (*ocistore.OCIStore)(nil)

func GetStore() KOTSStore {
	if !hasStore {
		globalStore = storeFromEnv()
	}

	return globalStore
}

func storeFromEnv() KOTSStore {
	storageBaseURI := os.Getenv("STORAGE_BASEURI")
	if storageBaseURI == "" {
		// KOTS 1.15 and earlier only supported s3 and there was no configuration
		storageBaseURI = fmt.Sprintf("s3://%s/%s", os.Getenv("S3_ENDPOINT"), os.Getenv("S3_BUCKET_NAME"))
	}

	parsedURI, err := url.Parse(storageBaseURI)
	if err != nil {
		panic(err) // store is critical
	}

	if parsedURI.Scheme == "docker" {
		return ocistore.StoreFromEnv()
	} else if parsedURI.Scheme == "s3" {
		return s3pg.S3PGStore{}
	}

	panic("unknown uri schema in store")
}
