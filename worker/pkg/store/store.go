package store

import (
	"database/sql"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	_ "github.com/lib/pq" // driver
	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/worker/pkg/config"
)

type SQLStore struct {
	db *sql.DB
	s3 *s3.S3
	c  *config.Config
}

func NewSQLStore(c *config.Config) (Store, error) {
	db, err := sql.Open("postgres", c.PostgresURI)
	if err != nil {
		return nil, errors.Wrap(err, "open db conn")
	}
	sess, err := session.NewSession(getS3Config(c))
	if err != nil {
		return nil, errors.Wrap(err, "new aws session")
	}
	return &SQLStore{db: db, s3: s3.New(sess), c: c}, nil
}
