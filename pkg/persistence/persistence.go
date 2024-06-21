package persistence

import (
	"fmt"
	"os"

	"github.com/rqlite/gorqlite"
)

var db *gorqlite.Connection

func IsInitialized() bool {
	return db != nil
}

func SetDB(database *gorqlite.Connection) {
	db = database
}

func MustGetDBSession() *gorqlite.Connection {
	if db != nil {
		return db
	}
	newDB, err := gorqlite.Open(os.Getenv("RQLITE_URI"))
	if err != nil {
		fmt.Printf("error connecting to rqlite: %v\n", err)
		panic(err)
	}
	db = &newDB
	return db
}
