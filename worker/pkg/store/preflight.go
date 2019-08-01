package store

import (
	"context"

	"github.com/pkg/errors"
)

func (s *SQLStore) SetWatchVersionPreflightSpec(ctx context.Context, watchID string, sequence int, preflightSpec string) error {
	query := `insert into preflight_spec (watch_id, sequence, spec) values ($1, $2, $3)`
	_, err := s.db.ExecContext(ctx, query, watchID, sequence, preflightSpec)
	if err != nil {
		return errors.Wrap(err, "exec query")
	}

	return nil
}
