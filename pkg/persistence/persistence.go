package persistence

import (
	"database/sql"
)

var (
	db  *sql.DB
	uri string
)

func IsPostgres() bool {
	return uri != ""
}

func SetDB(database *sql.DB) {
	db = database
}

func InitDB(databaseUri string) {
	uri = databaseUri
	MustGetDBSession()
}
