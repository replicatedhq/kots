package persistence

import (
	"database/sql"
)

var (
	PostgresURI string
	SQLiteURI   string
	mockDB      *sql.DB
)

func MustGetDBSession() *sql.DB {
	if mockDB != nil {
		return mockDB
	}

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

func InitMockDB(mock *sql.DB) {
	mockDB = mock
}
