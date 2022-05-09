//go:build !testing

package persistence

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

func MustGetDBSession() *sql.DB {
	if db != nil {
		return db
	}
	newDB, err := sql.Open("postgres", uri)
	if err != nil {
		fmt.Printf("error connecting to postgres: %v\n", err)
		panic(err)
	}
	db = newDB
	return db
}
