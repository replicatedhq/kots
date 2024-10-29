package store

import (
	"github.com/replicatedhq/kots/pkg/util"
)

var (
	globalStore Store
)

func GetStore() Store {
	if util.IsUpgradeService() {
		panic("store cannot not be used in the upgrade service")
	}
	if globalStore == nil {
		panic("store not initialized")
	}
	return globalStore
}

func SetStore(s Store) {
	if s == nil {
		globalStore = nil
		return
	}
	globalStore = s
}
