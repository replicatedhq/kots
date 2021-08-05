package persistence

import (
	"database/sql"
)

var (
	PostgresURI string
	SQLiteURI   string
)

func MustGetDBSession() *sql.DB {
	if SQLiteURI != "" {
		return mustGetSQLiteSession()
	}

	return mustGetPGSession()
}

func IsSQlite() bool {
	return SQLiteURI != ""
}

func IsPostgres() bool {
	return PostgresURI != ""
}
