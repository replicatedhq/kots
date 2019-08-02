package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
)

func (s *SQLStore) SetWatchTroubleshootCollectors(ctx context.Context, watchID string, collectors []byte) error {
	query := `insert into watch_troubleshoot_collector (watch_id, release_collector, release_collector_updated_at) values ($1, $2, $3)
	on conflict (watch_id) do update set release_collector = EXCLUDED.release_collector, release_collector_updated_at = EXCLUDED.release_collector_updated_at`
	_, err := s.db.ExecContext(ctx, query, watchID, collectors, time.Now())

	return err
}

func (s *SQLStore) SetWatchTroubleshootAnalyzers(ctx context.Context, watchID string, collectors []byte) error {
	query := `insert into watch_troubleshoot_analyzer (watch_id, release_analyzer, release_analyzer_updated_at) values ($1, $2, $3)
	on conflict (watch_id) do update set release_analyzer = EXCLUDED.release_analyzer, release_analyzer_updated_at = EXCLUDED.release_analyzer_updated_at`
	_, err := s.db.ExecContext(ctx, query, watchID, collectors, time.Now())

	return err
}

func (s *SQLStore) GetAnalyzeSpec(ctx context.Context, watchID string) (string, error) {
	query := `select release_analyzer, updated_analyzer, release_analyzer_updated_at, updated_analyzer_updated_at, use_updated_analyzer from
	watch_troubleshoot_analyzer where watch_id = $1`
	row := s.db.QueryRowContext(ctx, query, watchID)

	var releaseAnalyzer string
	var releaseAnalyzerUpdatedAt time.Time
	var updatedAnalyzer sql.NullString
	var updatedAnalyzerUpdatedAt pq.NullTime
	var useUpdatedAnalyzer sql.NullBool

	if err := row.Scan(&releaseAnalyzer, &updatedAnalyzer, &releaseAnalyzerUpdatedAt, &updatedAnalyzerUpdatedAt, &useUpdatedAnalyzer); err != nil {
		return "", err
	}

	if !updatedAnalyzer.Valid || !updatedAnalyzerUpdatedAt.Valid {
		return releaseAnalyzer, nil
	}

	if useUpdatedAnalyzer.Valid && useUpdatedAnalyzer.Bool {
		return updatedAnalyzer.String, nil
	}

	if releaseAnalyzerUpdatedAt.After(updatedAnalyzerUpdatedAt.Time) {
		return releaseAnalyzer, nil
	}

	return updatedAnalyzer.String, nil
}

func (s *SQLStore) GetTroubleshootSpec(ctx context.Context, watchID string) (string, error) {
	// TODO: supply troubleshoot with a spec.  It has its own default for now.
	return "", nil
}
