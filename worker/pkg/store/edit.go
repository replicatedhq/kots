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
	shipEditQuery := `select ship_edit.id, ship_edit.watch_id, ship_edit.user_id, ship_edit.result,
	ship_edit.created_at, ship_edit.finished_at, ship_edit.is_headless, watch.current_state
	from ship_edit
	inner join watch on watch.id = ship_edit.watch_id
	where ship_edit.id = $1`
	row := s.db.QueryRowContext(ctx, shipEditQuery, editID)

	editSession := types.EditSession{}
	var finishedAt pq.NullTime
	var result sql.NullString
	var isHeadless sql.NullBool

	err := row.Scan(&editSession.ID, &editSession.WatchID, &editSession.UserID, &result,
		&editSession.CreatedAt, &finishedAt, &isHeadless, &editSession.StateJSON)
	if err != nil {
		return nil, errors.Wrap(err, "scan ship_edit")
	}

	if finishedAt.Valid {
		editSession.FinishedAt = finishedAt.Time
	}

	if result.Valid {
		editSession.Result = result.String
	}

	if isHeadless.Valid {
		editSession.IsHeadless = isHeadless.Bool
	} /* edit default to false */

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
