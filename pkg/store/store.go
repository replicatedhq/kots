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
		hasStore = true
	}

	return globalStore
}

func storeFromEnv() Store {
	return kotsstore.StoreFromEnv()
}

func SetStore(s Store) {
	if s == nil {
		hasStore = false
		globalStore = nil
		return
	}
	hasStore = true
	globalStore = s
}
