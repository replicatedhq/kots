package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
)

func (s *SQLStore) ListReadyImageChecks(ctx context.Context) ([]string, error) {
	query := `update image_watch
		  set started_processing_at = $1
		  where id in (
		    select id from image_watch
		    where checked_at is null and started_processing_at is null
		    limit 10
		  )
		returning id`
	rows, err := s.db.QueryContext(ctx, query, time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}
	defer rows.Close()

	var imageCheckIDs []string
	for rows.Next() {
		var imageCheckID string
		if err := rows.Scan(&imageCheckID); err != nil {
			return imageCheckIDs, errors.Wrap(err, "rows scan")
		}
		imageCheckIDs = append(imageCheckIDs, imageCheckID)
	}

	return imageCheckIDs, nil
}

func (s *SQLStore) GetImageCheck(ctx context.Context, imageCheckID string) (*types.ImageCheck, error) {
	query := `select image_name, checked_at, is_private, versions_behind, detected_version,
		latest_version, compatible_version, check_error from image_watch where id = $1`
	row := s.db.QueryRowContext(ctx, query, imageCheckID)

	imageCheck := types.ImageCheck{
		ID: imageCheckID,
	}

	var checkedAt pq.NullTime
	var versionsBehind sql.NullInt64
	var detectedVersion sql.NullString
	var latestVersion sql.NullString
	var compatibleVersion sql.NullString
	var checkError sql.NullString
	if err := row.Scan(&imageCheck.Name, &checkedAt, &imageCheck.IsPrivate, &versionsBehind, &detectedVersion,
		&latestVersion, &compatibleVersion, &checkError); err != nil {
		return nil, errors.Wrap(err, "scan")
	}

	if checkedAt.Valid {
		imageCheck.CheckedAt = checkedAt.Time
	}
	if versionsBehind.Valid {
		imageCheck.VersionsBehind = versionsBehind.Int64
	}
	if detectedVersion.Valid {
		imageCheck.DetectedVersion = detectedVersion.String
	}
	if latestVersion.Valid {
		imageCheck.LatestVersion = latestVersion.String
	}
	if compatibleVersion.Valid {
		imageCheck.CompatibleVersion = compatibleVersion.String
	}
	if checkError.Valid {
		imageCheck.CheckError = checkError.String
	}

	return &imageCheck, nil
}

func (s *SQLStore) UpdateImageCheck(ctx context.Context, imageCheck *types.ImageCheck) error {
	query := `update image_watch set checked_at = $1, is_private = $2, versions_behind = $3,
		detected_version = $4, latest_version = $5, compatible_version = $6, check_error = $7,
		path = $8 where id = $9`
	_, err := s.db.ExecContext(ctx, query, time.Now(), imageCheck.IsPrivate, imageCheck.VersionsBehind,
		imageCheck.DetectedVersion, imageCheck.LatestVersion, imageCheck.CompatibleVersion,
		imageCheck.CheckError, imageCheck.Path, imageCheck.ID)

	return err
}
