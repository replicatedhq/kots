//go:build testing
// +build testing

package persistence

import "github.com/rqlite/gorqlite"

func MustGetDBSession() *gorqlite.Connection {
	return db
}
