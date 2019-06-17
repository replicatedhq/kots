package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
)

func (s *SQLStore) ListReadyUpdateIDs(ctx context.Context) ([]string, error) {
	maxTime := time.Now().Add(0 - time.Duration(time.Minute*5))

	query := `select id, started_at from ship_update where finished_at is null`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}
	defer rows.Close()

	var watchIDs []string
	for rows.Next() {
		var watchID string
		var startedAt pq.NullTime

		if err := rows.Scan(&watchID, &startedAt); err != nil {
			return watchIDs, errors.Wrap(err, "rows scan")
		}

		if !startedAt.Valid {
			watchIDs = append(watchIDs, watchID)
		} else {
			if startedAt.Time.Before(maxTime) {
				watchIDs = append(watchIDs, watchID)
			}
		}
	}
	return watchIDs, rows.Err()
}

func (s *SQLStore) SetUpdateStarted(ctx context.Context, updateID string) error {
	query := `update ship_update set started_at = $1 where id = $2`
	_, err := s.db.ExecContext(ctx, query, time.Now(), updateID)
	return err
}

func (s *SQLStore) GetUpdate(ctx context.Context, updateID string) (*types.UpdateSession, error) {
	shipUpdateQuery := `select ship_update.id, ship_update.watch_id, ship_update.result, ship_update.user_id,
	ship_update.created_at, ship_update.finished_at, watch.current_state
	from ship_update
	inner join watch on watch.id = ship_update.watch_id
	where ship_update.id = $1`
	row := s.db.QueryRowContext(ctx, shipUpdateQuery, updateID)

	updateSession := types.UpdateSession{}
	var finishedAt pq.NullTime
	var result sql.NullString
	var userID sql.NullString

	err := row.Scan(&updateSession.ID, &updateSession.WatchID, &result, &userID,
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

	if userID.Valid {
		updateSession.UserID = userID.String
	}

	nextSequence, err := s.GetNextUploadSequence(ctx, updateSession.WatchID)
	if err != nil {
		return nil, errors.Wrap(err, "get next upload sequence")
	}
	updateSession.UploadSequence = nextSequence

	return &updateSession, nil
}

func (s *SQLStore) SetUpdateStatus(ctx context.Context, watchID string, status string) error {
	query := `update ship_update set result = $1, finished_at = $2 where id = $3`
	_, err := s.db.ExecContext(ctx, query, status, time.Now(), watchID)
	return err
}
