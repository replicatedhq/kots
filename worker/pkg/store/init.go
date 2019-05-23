package store

import (
	"context"
	"database/sql"
	"math/rand"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (s *SQLStore) GetInit(ctx context.Context, initID string) (*types.InitSession, error) {
	query := `select
			ship_init.id, ship_init.upstream_uri, ship_init.created_at, ship_init.finished_at,
			ship_init.result, ship_init.user_id, ship_init.cluster_id, ship_init.github_path,
			ship_init.requested_upstream_uri
		from ship_init
		where ship_init.id = $1`
	row := s.db.QueryRowContext(ctx, query, initID)

	initSession := types.InitSession{}
	var finishedAt pq.NullTime
	var result sql.NullString
	var clusterID sql.NullString
	var githubPath sql.NullString
	var requestedUpstreamURI sql.NullString

	if err := row.Scan(&initSession.ID, &initSession.UpstreamURI, &initSession.CreatedAt,
		&finishedAt, &result, &initSession.UserID, &clusterID,
		&githubPath, &requestedUpstreamURI); err != nil {
		return nil, errors.Wrap(err, "scan")
	}
	if finishedAt.Valid {
		initSession.FinishedAt = finishedAt.Time
	}
	if result.Valid {
		initSession.Result = result.String
	}
	if clusterID.Valid {
		initSession.ClusterID = clusterID.String
	}
	if githubPath.Valid {
		initSession.GitHubPath = githubPath.String
	}
	if requestedUpstreamURI.Valid {
		initSession.RequestedUpstreamURI = requestedUpstreamURI.String
	}

	user, err := s.GetUser(ctx, initSession.UserID)
	if err != nil {
		return nil, errors.Wrap(err, "get user")
	}
	initSession.Username = user.GetUsername()

	return &initSession, nil
}

func (s *SQLStore) SetInitStatus(ctx context.Context, initID string, status string) error {
	query := `update ship_init set result = $1, finished_at = $2 where id = $3`
	_, err := s.db.ExecContext(ctx, query, status, time.Now(), initID)
	return err
}

func (s *SQLStore) CreateWatchFromState(ctx context.Context, stateJSON []byte, title string, iconURI string, slug string, userID string, initID string, clusterID string, githubPath string, parentWatchID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return errors.Wrap(err, "begin transaction")
	}
	defer tx.Rollback()

	query := ""

	if parentWatchID == "" {
		query = `insert into watch (id, current_state, title, icon_uri, created_at, slug, parent_watch_id)
	values ($1, $2, $3, $4, $5, $6, NULL)`
		_, err = tx.ExecContext(ctx, query, initID, stateJSON, title, iconURI, time.Now(), slug)
		if err != nil {
			return errors.Wrap(err, "create watch no parent")
		}
	} else {
		query = `insert into watch (id, current_state, title, icon_uri, created_at, slug, parent_watch_id)
	values ($1, $2, $3, $4, $5, $6, $7)`
		_, err = tx.ExecContext(ctx, query, initID, stateJSON, title, iconURI, time.Now(), slug, parentWatchID)
		if err != nil {
			return errors.Wrap(err, "create watch")
		}
	}

	query = `insert into user_watch (user_id, watch_id) values ($1, $2)`
	_, err = tx.ExecContext(ctx, query, userID, initID)
	if err != nil {
		return errors.Wrap(err, "create user-watch")
	}

	if clusterID != "" {
		query = `insert into watch_cluster (watch_id, cluster_id, github_path) values ($1, $2, $3)`
		_, err := tx.ExecContext(ctx, query, initID, clusterID, githubPath)
		if err != nil {
			return errors.Wrap(err, "create cluster-watch")
		}
	}

	webhookNotificationID := types.GenerateID()
	query = `insert into ship_notification (id, watch_id, created_at, enabled) values ($1, $2, $3, $4)`
	_, err = tx.ExecContext(ctx, query, webhookNotificationID, initID, time.Now(), 1)
	if err != nil {
		return errors.Wrap(err, "create webhook notification id")
	}
	query = `insert into webhook_notification (notification_id, destination_uri, created_at) values ($1, $2, $3)`
	_, err = tx.ExecContext(ctx, query, webhookNotificationID, "placeholder", time.Now())
	if err != nil {
		return errors.Wrap(err, "create webhook notification")
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "commit transaction")
	}

	return nil
}
