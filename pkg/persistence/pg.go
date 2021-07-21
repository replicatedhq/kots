package persistence

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

var postgresDB *sql.DB

func mustGetPGSession() *sql.DB {
	if postgresDB != nil {
		return postgresDB
	}

	db, err := sql.Open("postgres", PostgresURI)
	if err != nil {
		fmt.Printf("error connecting to postgres: %v\n", err)
		panic(err)
	}

	postgresDB = db
	return db
}
