package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
)

func (s *SQLStore) GetUpdate(ctx context.Context, updateID string) (*types.UpdateSession, error) {
	shipUpdateQuery := `select ship_update.id, ship_update.watch_id, ship_update.user_id, ship_update.result,
	ship_update.created_at, ship_update.finished_at, watch.current_state
	from ship_update
	inner join watch on watch.id = ship_update.watch_id
	where ship_update.id = $1`
	row := s.db.QueryRowContext(ctx, shipUpdateQuery, updateID)

	updateSession := types.UpdateSession{}
	var finishedAt pq.NullTime
	var result sql.NullString

	err := row.Scan(&updateSession.ID, &updateSession.WatchID, &updateSession.UserID, &result,
		&updateSession.CreatedAt, &finishedAt, &updateSession.StateJSON)
	if err != nil {
		return nil, errors.Wrap(err, "scan ship_update")
	}

	if finishedAt.Valid {
		updateSession.FinishedAt = finishedAt.Time
	}

	if result.Valid {
		updateSession.Result = result.String
	}

	nextSequence, err := s.GetNextUploadSequence(ctx, updateSession.WatchID)
	if err != nil {
		return nil, errors.Wrap(err, "get next upload sequence")
	}
	updateSession.UploadSequence = nextSequence

	return &updateSession, nil
}

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

func (s *SQLStore) SetUpdateStatus(ctx context.Context, watchID string, status string) error {
	query := `update ship_update set result = $1, finished_at = $2 where id = $3`
	_, err := s.db.ExecContext(ctx, query, status, time.Now(), watchID)
	return err
}

func (s *SQLStore) UpdateWatchFromState(ctx context.Context, watchID string, stateJSON []byte) error {
	query := `update watch set current_state = $1, updated_at = $2 where id = $3`
	_, err := s.db.ExecContext(ctx, query, stateJSON, time.Now(), watchID)
	if err != nil {
		return errors.Wrap(err, "update watch")
	}

	return nil
}
