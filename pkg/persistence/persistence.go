package persistence

import "github.com/rqlite/gorqlite"

var (
	db  *gorqlite.Connection
	uri string
)

func IsInitialized() bool {
	return db != nil
}

func SetDB(database *gorqlite.Connection) {
	db = database
}

func InitDB(databaseUri string) {
	uri = databaseUri
	MustGetDBSession()
}
