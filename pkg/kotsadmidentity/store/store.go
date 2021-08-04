package store

import (
	"path/filepath"

	"github.com/replicatedhq/kots/pkg/persistence"
)

var (
	hasStore    = false
	globalStore DexStore
)

var _ DexStore = (*PostgresStore)(nil)
var _ DexStore = (*SQLiteStore)(nil)

func GetStore() DexStore {
	if hasStore {
		return globalStore
	}

	hasStore = true
	if persistence.IsSQlite() {
		globalStore = &SQLiteStore{
			dbFilename: filepath.Join(filepath.Dir(persistence.SQLiteURI), "dex.db"),
		}
	} else {
		globalStore = &PostgresStore{}
	}

	return globalStore
}
