package store

var (
	hasStore    = false
	globalStore DexStore
)

func GetStore() DexStore {
	if hasStore {
		return globalStore
	}

	hasStore = true
	globalStore = &PostgresStore{}

	return globalStore
}
