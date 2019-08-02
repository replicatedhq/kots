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
	query := `select id, upstream_uri, created_at, finished_at, result, user_id, cluster_id, github_path,
		requested_upstream_uri, parent_watch_id, parent_sequence
		from ship_init where id = $1`
	row := s.db.QueryRowContext(ctx, query, initID)

	initSession := types.InitSession{}
	var finishedAt pq.NullTime
	var result sql.NullString
	var clusterID sql.NullString
	var githubPath sql.NullString
	var requestedUpstreamURI sql.NullString
	var parentWatchID sql.NullString
	var parentSequence sql.NullInt64

	if err := row.Scan(&initSession.ID, &initSession.UpstreamURI, &initSession.CreatedAt,
		&finishedAt, &result, &initSession.UserID, &clusterID,
		&githubPath, &requestedUpstreamURI, &parentWatchID, &parentSequence); err != nil {
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
	if parentWatchID.Valid {
		s := parentWatchID.String
		initSession.ParentWatchID = &s
	}
	if parentSequence.Valid {
		seq := int(parentSequence.Int64)
		initSession.ParentSequence = &seq
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

func (s *SQLStore) CreateWatchFromState(ctx context.Context, stateJSON []byte, metadata []byte, title string, iconURI string, slug string, userID string, initID string, clusterID string, githubPath string, parentWatchID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return errors.Wrap(err, "begin transaction")
	}
	defer tx.Rollback()

	query := ""

	if parentWatchID == "" {
		query = `insert into watch (id, current_state, title, icon_uri, created_at, slug, parent_watch_id, metadata)
	values ($1, $2, $3, $4, $5, $6, NULL, $7)`
		_, err = tx.ExecContext(ctx, query, initID, stateJSON, title, iconURI, time.Now(), slug, metadata)
		if err != nil {
			return errors.Wrap(err, "create watch no parent")
		}
	} else {
		query = `insert into watch (id, current_state, title, icon_uri, created_at, slug, parent_watch_id, metadata)
	values ($1, $2, $3, $4, $5, $6, $7, $8)`
		_, err = tx.ExecContext(ctx, query, initID, stateJSON, title, iconURI, time.Now(), slug, parentWatchID, metadata)
		if err != nil {
			return errors.Wrap(err, "create watch")
		}

		query = `
      SELECT ship_user.id as contributor_id
      FROM user_watch
        JOIN ship_user ON ship_user.id = user_watch.user_id
      WHERE user_watch.watch_id = $1 AND ship_user.id != $2
		`
		contributors, err := tx.QueryContext(ctx, query, parentWatchID, userID)
		if err != nil {
			return errors.Wrap(err, "create watch get contributors")
		}

		var contributorIDs []string

		for contributors.Next() {
			var contributorID string
			err := contributors.Scan(&contributorID)
			if err != nil {
				return errors.Wrap(err, "scan contributor row")
			}

			contributorIDs = append(contributorIDs, contributorID)
		}

		err = contributors.Err()
		if err != nil {
			return errors.Wrap(err, "scan contributor row loop")
		}

		for _, id := range contributorIDs {
			query = `INSERT into user_watch (user_id, watch_id) VALUES ($1, $2)`

			_, err = tx.ExecContext(ctx, query, id, initID)
			if err != nil {
				return errors.Wrap(err, "create watch insert_contributors")
			}
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
