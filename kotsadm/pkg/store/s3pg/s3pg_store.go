package s3pg

import (
	"database/sql"

	"github.com/pkg/errors"
)

type S3PGStore struct {
}

func (s S3PGStore) IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errors.Cause(err) == sql.ErrNoRows
}
