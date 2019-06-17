package store

import (
	"database/sql"

	_ "github.com/lib/pq" // driver
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
)

type SQLStore struct {
	db *sql.DB
	c  *config.Config
}

func NewSQLStore(c *config.Config) (Store, error) {
	db, err := sql.Open("postgres", c.PostgresURI)
	if err != nil {
		return nil, errors.Wrap(err, "open db conn")
	}
	return &SQLStore{db: db, c: c}, nil
}
