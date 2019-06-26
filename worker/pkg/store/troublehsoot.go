package store

import (
	"context"
	"time"
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
