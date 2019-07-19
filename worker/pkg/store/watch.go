package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
)

var (
	watchPollInterval = time.Minute * 15
)

func (s *SQLStore) GetParentWatchID(ctx context.Context, watchID string) (*string, error) {
	query := `select parent_watch_id from watch where id = $1`
	row := s.db.QueryRowContext(ctx, query, watchID)

	var parentWatchID sql.NullString
	if err := row.Scan(&parentWatchID); err != nil {
		return nil, errors.Wrap(err, "row scan")
	}

	if !parentWatchID.Valid {
		return nil, nil
	}

	var str = parentWatchID.String
	return &str, nil
}

// ListReadyWatchIDs will return a list of midstream watches that need to be updated
func (s *SQLStore) ListReadyWatchIDs(ctx context.Context) ([]string, error) {
	var maxDate = time.Now().Add(0 - watchPollInterval)

	query := `select id from watch where last_watch_check_at < $1 or last_watch_check_at is null and parent_watch_id is null order by updated_at desc`
	rows, err := s.db.QueryContext(ctx, query, maxDate)
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}
	defer rows.Close()

	var watchIDs []string
	for rows.Next() {
		var watchID string
		if err := rows.Scan(&watchID); err != nil {
			return watchIDs, errors.Wrap(err, "rows scan")
		}
		watchIDs = append(watchIDs, watchID)
	}
	return watchIDs, rows.Err()
}

func (s SQLStore) ListDownstreamWatchIDs(ctx context.Context, watchID string) ([]string, error) {
	query := `select id from watch where parent_watch_id = $1`
	rows, err := s.db.QueryContext(ctx, query, watchID)
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}
	defer rows.Close()

	var watchIDs []string
	for rows.Next() {
		var watchID string
		if err := rows.Scan(&watchID); err != nil {
			return watchIDs, errors.Wrap(err, "rows scan")
		}
		watchIDs = append(watchIDs, watchID)
	}
	return watchIDs, rows.Err()
}

func (s *SQLStore) GetPullRequestNotification(ctx context.Context, notificationID string) (*types.PullRequestNotification, error) {
	query := `select org, repo, branch, root_path, github_installation_id from pullrequest_notification where notification_id = $1`
	row := s.db.QueryRowContext(ctx, query, notificationID)

	pullRequestNotification := types.PullRequestNotification{}

	var branch sql.NullString
	var rootPath sql.NullString
	if err := row.Scan(&pullRequestNotification.Org, &pullRequestNotification.Repo, &branch, &rootPath, &pullRequestNotification.GithubInstallationID); err != nil {
		return nil, errors.Wrap(err, "scan")
	}

	if branch.Valid {
		pullRequestNotification.Branch = branch.String
	}
	if rootPath.Valid {
		pullRequestNotification.RootPath = rootPath.String
	}

	return &pullRequestNotification, nil
}

func (s *SQLStore) GetEmailNotification(ctx context.Context, notificationID string) (*types.EmailNotification, error) {
	query := `select recipient from email_notification where notification_id = $1`
	row := s.db.QueryRowContext(ctx, query, notificationID)

	emailNotification := types.EmailNotification{}

	if err := row.Scan(&emailNotification.Address); err != nil {
		return nil, errors.Wrap(err, "scan")
	}

	return &emailNotification, nil
}

func (s *SQLStore) GetWebhookNotification(ctx context.Context, notificationID string) (*types.WebhookNotification, error) {
	query := `select destination_uri from webhook_notification where notification_id = $1`
	row := s.db.QueryRowContext(ctx, query, notificationID)

	webhookNotification := types.WebhookNotification{}

	if err := row.Scan(&webhookNotification.URI); err != nil {
		return nil, errors.Wrap(err, "scan")
	}

	return &webhookNotification, nil
}

func (s *SQLStore) GetWatchIDFromSlug(ctx context.Context, slug string, userID string) (string, error) {
	query := `select watch_id from user_watch inner join watch on watch.id = user_watch.watch_id where watch.slug = $1 and user_watch.user_id = $2`
	row := s.db.QueryRowContext(ctx, query, slug, userID)

	var watchID string

	if err := row.Scan(&watchID); err != nil {
		return "", errors.Wrap(err, "scan")
	}

	return watchID, nil
}

func (s *SQLStore) GetWatch(ctx context.Context, watchID string) (*types.Watch, error) {
	query := `select id, title, current_state, created_at, updated_at from watch where id = $1`
	row := s.db.QueryRowContext(ctx, query, watchID)

	watch := types.Watch{}
	var updatedAt pq.NullTime
	if err := row.Scan(&watch.ID, &watch.Title, &watch.StateJSON, &watch.CreatedAt, &updatedAt); err != nil {
		return nil, errors.Wrap(err, "scan")
	}
	if updatedAt.Valid {
		watch.UpdatedAt = updatedAt.Time
	}

	watch.Notifications = make([]*types.WatchNotification, 0, 0)

	var enabled sql.NullBool

	// Get email notifications, turn these into webhooks
	query = `select n.id, n.enabled, e.recipient from email_notification e inner join ship_notification n on n.id = e.notification_id where n.id in (select id from ship_notification where watch_id = $1)`
	rows, err := s.db.QueryContext(ctx, query, watchID)
	if err != nil {
		return nil, errors.Wrap(err, "query email_notification")
	}
	defer rows.Close()
	for rows.Next() {
		notification := types.WatchNotification{
			Email: &types.EmailNotification{},
		}
		if err := rows.Scan(&notification.ID, &enabled, &notification.Email.Address); err != nil {
			return nil, errors.Wrap(err, "scan")
		}
		notification.Enabled = !enabled.Valid || enabled.Bool

		watch.Notifications = append(watch.Notifications, &notification)
	}

	// Get webhook notifications
	query = `select n.id, n.enabled, w.destination_uri from webhook_notification w inner join ship_notification n on n.id = w.notification_id where n.id in (select id from ship_notification where watch_id = $1)`
	rows, err = s.db.QueryContext(ctx, query, watchID)
	if err != nil {
		return nil, errors.Wrap(err, "query webhook_notification")
	}
	defer rows.Close()
	for rows.Next() {
		notification := types.WatchNotification{
			Webhook: &types.WebhookNotification{},
		}
		if err := rows.Scan(&notification.ID, &enabled, &notification.Webhook.URI); err != nil {
			return nil, errors.Wrap(err, "scan")
		}
		notification.Enabled = !enabled.Valid || enabled.Bool

		notification.Webhook.Secret = "not-implemented"

		watch.Notifications = append(watch.Notifications, &notification)
	}

	// Get pull request notifications
	query = `select n.id, n.enabled, p.org, p.repo, p.branch, p.root_path, p.github_installation_id from pullrequest_notification p inner join ship_notification n on n.id = p.notification_id where n.id in (select id from ship_notification where watch_id = $1)`
	rows, err = s.db.QueryContext(ctx, query, watchID)
	if err != nil {
		return nil, errors.Wrap(err, "query pullrequest_notification")
	}
	defer rows.Close()
	for rows.Next() {
		notification := types.WatchNotification{
			PullRequest: &types.PullRequestNotification{},
		}
		var branch sql.NullString
		var rootPath sql.NullString
		if err := rows.Scan(&notification.ID, &enabled, &notification.PullRequest.Org, &notification.PullRequest.Repo, &branch, &rootPath, &notification.PullRequest.GithubInstallationID); err != nil {
			return nil, errors.Wrap(err, "scan")
		}
		notification.Enabled = !enabled.Valid || enabled.Bool

		if branch.Valid {
			notification.PullRequest.Branch = branch.String
		}
		if rootPath.Valid {
			notification.PullRequest.RootPath = rootPath.String
		}

		watch.Notifications = append(watch.Notifications, &notification)
	}

	return &watch, nil
}

func (s *SQLStore) GetNotificationWatchID(ctx context.Context, notificationID string) (string, error) {
	query := `
select watch_id from ship_notification where id = $1
`
	row := s.db.QueryRowContext(ctx, query, notificationID)

	var watchID string
	err := row.Scan(&watchID)
	if err != nil {
		return "", errors.Wrap(err, "scan getNotificationWatchID")
	}

	return watchID, nil
}

func (s *SQLStore) GetWatches(ctx context.Context, userID string) ([]*types.Watch, error) {
	query := `select
user_id, watch_id as id, watch.title, watch.slug, watch.current_state, watch.created_at, watch.updated_at
from user_watch
join watch on watch.id = user_watch.watch_id
where user_watch.user_id = $1
order by watch.created_at`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}
	defer rows.Close()

	var watches []*types.Watch
	for rows.Next() {
		watch := types.Watch{}
		var updatedAt pq.NullTime

		if err := rows.Scan(&userID, &watch.ID, &watch.Title, &watch.Slug, &watch.StateJSON, &watch.CreatedAt, &updatedAt); err != nil {
			return watches, errors.Wrap(err, "rows scan")
		}
		watches = append(watches, &watch)
	}
	return watches, rows.Err()
}

func (s *SQLStore) GetSequenceNumberForWatchID(ctx context.Context, watchID string) (int, error) {
	var currentSequence int
	getSequenceNumberQuery := `
    SELECT max(sof.sequence)
    FROM ship_output_files sof
    WHERE sof.watch_id = $1
	`
	row := s.db.QueryRowContext(ctx, getSequenceNumberQuery, watchID)
	if err := row.Scan(&currentSequence); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}

		return currentSequence, errors.Wrap(err, "scan ship_output_files")
	}
	return currentSequence, nil
}

func (s *SQLStore) GetSequenceNumberForNotificationID(ctx context.Context, notificationID string) (int, error) {
	var currentSequence int
	getSequenceNumberQuery := `
    SELECT max(sof.sequence)
    FROM ship_output_files sof
    INNER JOIN ship_notification sn ON sn.watch_id = sof.watch_id
    WHERE sn.id = $1
	`
	row := s.db.QueryRowContext(ctx, getSequenceNumberQuery, notificationID)
	if err := row.Scan(&currentSequence); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}

		return currentSequence, errors.Wrap(err, "scan ship_output_files")
	}
	return currentSequence, nil
}

func (s *SQLStore) CreateWatchUpdate(ctx context.Context, watchID string, parentSequence *int) error {
	id, err := uuid.GenerateUUID()
	if err != nil {
		return errors.Wrap(err, "generate uuid")
	}

	query := `insert into ship_update (id, watch_id, created_at, parent_sequence) values ($1, $2, $3, $4)`
	_, err = s.db.ExecContext(ctx, query, strings.Replace(id, "-", "", -1), watchID, time.Now(), parentSequence)
	if err != nil {
		return errors.Wrap(err, "create watch update")
	}

	return nil
}

func (s *SQLStore) CancelIncompleteWatchUpdates(ctx context.Context, watchID string) error {
	fmt.Printf("canceling incomplete updates for %s\n", watchID)
	query := `update ship_update set result = $1, finished_at = $2 where watch_id = $3 and finished_at is null`
	_, err := s.db.ExecContext(ctx, query, "aborted", time.Now(), watchID)
	return err
}

func (s *SQLStore) SetWatchChecked(ctx context.Context, watchID string) error {
	query := `update watch set last_watch_check_at = $1 where id = $2`
	_, err := s.db.ExecContext(ctx, query, time.Now(), watchID)
	if err != nil {
		return errors.Wrap(err, "set watch last checked")
	}

	return nil
}

func (s *SQLStore) SetWatchDeferred(ctx context.Context, watchID string) error {
	// this is a hack to back off the watch polling for now
	now := time.Now()

	query := `update watch set last_watch_check_at = $1 where id = $2`
	_, err := s.db.ExecContext(ctx, query, now.Add(time.Duration(watchPollInterval*3)), watchID)
	if err != nil {
		return errors.Wrap(err, "set watch last checked")
	}

	return nil
}

func (s *SQLStore) UpdateWatchState(ctx context.Context, watchID string, stateJSON []byte, metadata []byte) error {
	query := `update watch set current_state = $1, metadata = $2, updated_at = $3 where id = $4`
	_, err := s.db.ExecContext(ctx, query, stateJSON, metadata, time.Now(), watchID)
	if err != nil {
		return errors.Wrap(err, "update watch")
	}

	return nil
}

func (s *SQLStore) SetWatchLicense(ctx context.Context, watchID string, license []byte) error {
	query := `insert into watch_license (watch_id, license, license_updated_at) values ($1, $2, $3)
	on conflict (watch_id) do update set license = EXCLUDED.license, license_updated_at = EXCLUDED.license_updated_at`
	_, err := s.db.ExecContext(ctx, query, watchID, license, time.Now())
	if err != nil {
		return errors.Wrap(err, "set license")
	}

	return nil
}
