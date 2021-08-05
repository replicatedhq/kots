package store

import (
	"github.com/replicatedhq/kots/pkg/persistence"
)

var (
	hasStore    = false
	globalStore DexStore
)

var _ DexStore = (*PostgresStore)(nil)
var _ DexStore = (*K8sStore)(nil)

func GetStore() DexStore {
	if hasStore {
		return globalStore
	}

	hasStore = true
	if persistence.IsSQlite() {
		globalStore = &K8sStore{}
	} else {
		globalStore = &PostgresStore{}
	}

	return globalStore
}
