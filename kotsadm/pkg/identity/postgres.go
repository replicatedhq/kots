package identity

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
)

func createDexPostgresDatabase(database, user, password string) error {
	db := persistence.MustGetPGSession()

	databaseQ := pq.QuoteIdentifier(database)
	userQ := pq.QuoteIdentifier(user)
	passwordQ := pq.QuoteIdentifier(password)

	query := "SELECT 1 FROM pg_database WHERE datname = $1"
	row := db.QueryRow(query, database)
	var exists bool
	err := row.Scan(&exists)
	if err == sql.ErrNoRows {
		query := fmt.Sprintf("CREATE DATABASE %s", databaseQ)
		_, err := db.Exec(query)
		if err != nil {
			return errors.Wrap(err, "failed to create database")
		}
	} else if err != nil {
		return errors.Wrap(err, "failed to query database")
	}

	query = "SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = $1"
	row = db.QueryRow(query, user)
	err = row.Scan(&exists)
	if err == sql.ErrNoRows {
		query := fmt.Sprintf("CREATE USER %s", userQ)
		_, err := db.Exec(query)
		if err != nil {
			return errors.Wrap(err, "failed to create user")
		}
	} else if err != nil {
		return errors.Wrap(err, "failed to query user")
	}

	query = fmt.Sprintf(
		`ALTER USER %s WITH PASSWORD %s;
		GRANT ALL PRIVILEGES ON DATABASE %s TO %s;`,
		userQ, passwordQ, databaseQ, userQ,
	)
	_, err = db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}
