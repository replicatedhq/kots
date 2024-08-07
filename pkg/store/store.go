package store

import (
	"github.com/replicatedhq/kots/pkg/store/kotsstore"
	"github.com/replicatedhq/kots/pkg/util"
)

var (
	hasStore    = false
	globalStore Store
)

var _ Store = (*kotsstore.KOTSStore)(nil)

func GetStore() Store {
	if util.IsUpgradeService() {
		panic("store cannot not be used in the upgrade service")
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
