package store

import (
	"context"
	"database/sql"

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
