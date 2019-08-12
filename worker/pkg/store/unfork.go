package store

import (
	"context"
	"database/sql"
	"math/rand"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/worker/pkg/types"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (s *SQLStore) GetUnfork(ctx context.Context, unforkID string) (*types.UnforkSession, error) {
	query := `select
		ship_unfork.id, ship_unfork.upstream_uri, ship_unfork.fork_uri, ship_unfork.created_at,
		ship_unfork.finished_at, ship_unfork.result, ship_unfork.user_id, github_user.username
		from ship_unfork
		inner join ship_user on ship_user.id = ship_unfork.user_id
		inner join github_user on github_user.github_id = ship_user.github_id
		where ship_unfork.id = $1`
	row := s.db.QueryRowContext(ctx, query, unforkID)

	unforkSession := types.UnforkSession{}
	var finishedAt pq.NullTime
	var result sql.NullString

	if err := row.Scan(&unforkSession.ID, &unforkSession.UpstreamURI, &unforkSession.ForkURI,
		&unforkSession.CreatedAt, &finishedAt, &result, &unforkSession.UserID, &unforkSession.Username); err != nil {
		return nil, errors.Wrap(err, "scan")
	}
	if finishedAt.Valid {
		unforkSession.FinishedAt = finishedAt.Time
	}
	if result.Valid {
		unforkSession.Result = result.String
	}

	return &unforkSession, nil
}

func (s *SQLStore) SetUnforkStatus(ctx context.Context, unforkID string, status string) error {
	query := `update ship_unfork set result = $1, finished_at = $2 where id = $3`
	_, err := s.db.ExecContext(ctx, query, status, time.Now(), unforkID)
	return err
}
