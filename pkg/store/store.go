package store

import (
	"github.com/replicatedhq/kots/pkg/store/kotsstore"
	"github.com/replicatedhq/kots/pkg/store/ocistore"
)

var (
	hasStore    = false
	globalStore Store
)

var _ Store = (*kotsstore.KOTSStore)(nil)
var _ Store = (*ocistore.OCIStore)(nil)

func GetStore() Store {
	if !hasStore {
		globalStore = storeFromEnv()
		hasStore = true
	}

	return globalStore
}

func storeFromEnv() Store {
	return kotsstore.StoreFromEnv()
}
