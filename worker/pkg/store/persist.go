package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
)

func (s *SQLStore) GetNextUploadSequence(ctx context.Context, watchID string) (int, error) {
	var currentSequence int
	shipUpdateOutputFileSequenceQuery := `select max(ship_output_files.sequence) from ship_output_files where watch_id = $1 group by ship_output_files.watch_id`

	row := s.db.QueryRowContext(ctx, shipUpdateOutputFileSequenceQuery, watchID)
	if err := row.Scan(&currentSequence); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}

		return currentSequence, errors.Wrap(err, "scan ship_output_files")
	}
	return currentSequence + 1, nil
}

func (s *SQLStore) UpdateWatchFromState(ctx context.Context, watchID string, stateJSON []byte) error {
	query := `update watch set current_state = $1, updated_at = $2 where id = $3`
	_, err := s.db.ExecContext(ctx, query, stateJSON, time.Now(), watchID)
	if err != nil {
		return errors.Wrap(err, "update watch")
	}

	return nil
}
