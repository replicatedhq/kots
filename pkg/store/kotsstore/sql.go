package kotsstore

import "database/sql"

type scannable interface {
	Scan(dest ...interface{}) error
}

type queryable interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}
