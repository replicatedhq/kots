//go:build testing
// +build testing

package persistence

import (
	"database/sql"
)

func MustGetDBSession() *sql.DB {
	return db
}
