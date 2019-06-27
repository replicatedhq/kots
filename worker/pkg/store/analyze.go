package store

import (
	"context"

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

func (s *SQLStore) GetSupportBundle(ctx context.Context, supportBundleID string) (*types.SupportBundle, error) {
	query := `select id, status from supportbundle where id = $1`
	row := s.db.QueryRowContext(ctx, query, supportBundleID)

	supportBundle := types.SupportBundle{}
	if err := row.Scan(&supportBundle.ID, &supportBundle.Status); err != nil {
		return nil, errors.Wrap(err, "scan")
	}

	return &supportBundle, nil
}
