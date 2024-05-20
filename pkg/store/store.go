package store

import (
	"os"

	"github.com/replicatedhq/kots/pkg/store/kotsstore"
)

var (
	hasStore    = false
	globalStore Store
)

var _ Store = (*kotsstore.KOTSStore)(nil)

func GetStore() Store {
	if os.Getenv("IS_UPGRADE_SERVICE") == "true" {
		panic("store should not be used in the upgrade service")
	}
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
