package store

type DexStore interface {
	CreateDexDatabase(database string, user string, password string) error
	DatabaseUserExists(user string) (bool, error)
}
