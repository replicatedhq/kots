//go:build !testing

package persistence

import (
	"fmt"

	"github.com/rqlite/gorqlite"
)

func MustGetDBSession() *gorqlite.Connection {
	if db != nil {
		return db
	}
	newDB, err := gorqlite.Open(uri)
	if err != nil {
		fmt.Printf("error connecting to rqlite: %v\n", err)
		panic(err)
	}

	db = &newDB
	return db
}
