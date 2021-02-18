package store

import (
	"github.com/replicatedhq/kots/pkg/store/kotsstore"
)

var (
	hasStore    = false
	globalStore Store
)

var _ Store = (*kotsstore.KOTSStore)(nil)

func GetStore() Store {
	if !hasStore {
		globalStore = storeFromEnv()
	}

	return globalStore
}

func storeFromEnv() Store {
	// storageBaseURI := os.Getenv("STORAGE_BASEURI")
	// if storageBaseURI == "" {
	// 	// KOTS 1.15 and earlier only supported s3 and there was no configuration
	// 	storageBaseURI = fmt.Sprintf("s3://%s/%s", os.Getenv("S3_ENDPOINT"), os.Getenv("S3_BUCKET_NAME"))
	// }

	// parsedURI, err := url.Parse(storageBaseURI)
	// if err != nil {
	// 	panic(err) // store is critical
	// }

	// fmt.Printf("%s\n", parsedURI)

	return kotsstore.KOTSStore{}

}
