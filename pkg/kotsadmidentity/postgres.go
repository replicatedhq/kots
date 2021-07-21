package identity

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/persistence"
)

func postgresUserExists(user string) (bool, error) {
	db := persistence.MustGetDBSession()

	query := "SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = $1"
	row := db.QueryRow(query, user)

	var exists bool
	err := row.Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "failed to query user")
	}

	return true, nil
}

func CreateDexPostgresDatabase(database, user, password string) error {
	db := persistence.MustGetDBSession()

	databaseQ := pq.QuoteIdentifier(database)
	userQ := pq.QuoteIdentifier(user)

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

	exists, err = postgresUserExists(user)
	if err != nil {
		return errors.Wrap(err, "failed to query user")
	}

	if !exists {
		query := fmt.Sprintf("CREATE USER %s", userQ)
		_, err := db.Exec(query)
		if err != nil {
			return errors.Wrap(err, "failed to create user")
		}
	}

	query = fmt.Sprintf(
		`ALTER USER %s WITH PASSWORD '%s';
		GRANT ALL PRIVILEGES ON DATABASE %s TO %s;`,
		userQ, password, databaseQ, userQ,
	)
	_, err = db.Exec(query)
	if err != nil {
		return errors.Wrap(err, "failed to grant user privileges")
	}

	return nil
}
