package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
)

func (s *SQLStore) GetEdit(ctx context.Context, editID string) (*types.EditSession, error) {
	shipEditQuery := `select ship_update.id, ship_update.watch_id, ship_update.user_id, ship_update.result,
	ship_update.created_at, ship_update.finished_at, watch.current_state
	from ship_update
	inner join watch on watch.id = ship_update.watch_id
	where ship_update.id = $1`
	row := s.db.QueryRowContext(ctx, shipEditQuery, editID)

	editSession := types.EditSession{}
	var finishedAt pq.NullTime
	var result sql.NullString

	err := row.Scan(&editSession.ID, &editSession.WatchID, &editSession.UserID, &result,
		&editSession.CreatedAt, &finishedAt, &editSession.StateJSON)
	if err != nil {
		return nil, errors.Wrap(err, "scan ship_edit")
	}

	if finishedAt.Valid {
		editSession.FinishedAt = finishedAt.Time
	}

	if result.Valid {
		editSession.Result = result.String
	}

	nextSequence, err := s.GetNextUploadSequence(ctx, editSession.WatchID)
	if err != nil {
		return nil, errors.Wrap(err, "get next upload sequence")
	}
	editSession.UploadSequence = nextSequence

	return &editSession, nil
}

func (s *SQLStore) SetEditStatus(ctx context.Context, watchID string, status string) error {
	query := `update ship_edit set result = $1, finished_at = $2 where id = $3`
	_, err := s.db.ExecContext(ctx, query, status, time.Now(), watchID)
	return err
}
