package store

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
)

func (s *SQLStore) ListReadyAnalysisIDs(ctx context.Context) ([]string, error) {
	query := `select id from supportbundle where status = $1`
	rows, err := s.db.QueryContext(ctx, query, "uploaded")
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}
	defer rows.Close()

	var supportBundleIDs []string
	for rows.Next() {
		var supportBundleID string

		if err := rows.Scan(&supportBundleID); err != nil {
			return supportBundleIDs, errors.Wrap(err, "rows scan")
		}

		supportBundleIDs = append(supportBundleIDs, supportBundleID)
	}

	return supportBundleIDs, rows.Err()
}

func (s *SQLStore) SetAnalysisStarted(ctx context.Context, supportBundleID string) error {
	query := `update supportbundle set status = $1 where id = $2`
	_, err := s.db.ExecContext(ctx, query, "analyzing", supportBundleID)
	return err
}

func (s *SQLStore) SetAnalysisFailed(ctx context.Context, supportBundleID string) error {
	query := `update supportbundle set status = $1 where id = $2`
	_, err := s.db.ExecContext(ctx, query, "analysis_error", supportBundleID)
	return err
}

func (s *SQLStore) SetAnalysisSucceeded(ctx context.Context, supportBundleID string, insights string) error {
	id := types.GenerateID()
	query := `insert into supportbundle_analysis (id, supportbundle_id, error, max_severity, insights, created_at)
		values ($1, $2, null, null, $3, $4)`
	_, err := s.db.ExecContext(ctx, query, id, supportBundleID, insights, time.Now())
	return err
}

func (s *SQLStore) GetSupportBundle(ctx context.Context, supportBundleID string) (*types.SupportBundle, error) {
	query := `select id, status, watch_id from supportbundle where id = $1`
	row := s.db.QueryRowContext(ctx, query, supportBundleID)

	supportBundle := types.SupportBundle{}
	if err := row.Scan(&supportBundle.ID, &supportBundle.Status, &supportBundle.WatchID); err != nil {
		return nil, errors.Wrap(err, "scan")
	}

	return &supportBundle, nil
}
