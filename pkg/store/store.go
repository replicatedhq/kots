package store

import (
	"github.com/replicatedhq/kots/pkg/util"
)

var (
	hasStore    = false
	globalStore Store
)

func GetStore() Store {
	if util.IsUpgradeService() {
		panic("store cannot not be used in the upgrade service")
	}
	if !hasStore {
		panic("store not initialized")
	}
	return globalStore
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
