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
	db = newDB
	return db
}

// NewTransactionDBSession returns a dedicated connection for atomic
// multi-statement writes without changing the shared session's settings.
func NewTransactionDBSession() (*gorqlite.Connection, error) {
	conn, err := gorqlite.Open(os.Getenv("RQLITE_URI"))
	if err != nil {
		return nil, err
	}
	if err := conn.SetExecutionWithTransaction(true); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}
