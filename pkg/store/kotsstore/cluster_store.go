package kotsstore

import (
	"fmt"
	"time"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/rand"
	"github.com/rqlite/gorqlite"
	"go.uber.org/zap"
)

func (s *KOTSStore) ListClusters() ([]*downstreamtypes.Downstream, error) {
	db := persistence.MustGetDBSession()

	query := `select id, slug, title, snapshot_schedule, snapshot_ttl from cluster` // TODO the current sequence
	rows, err := db.QueryOne(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

	clusters := []*downstreamtypes.Downstream{}
	for rows.Next() {
		cluster := downstreamtypes.Downstream{}

		var snapshotSchedule gorqlite.NullString
		var snapshotTTL gorqlite.NullString

		if err := rows.Scan(&cluster.ClusterID, &cluster.ClusterSlug, &cluster.Name, &snapshotSchedule, &snapshotTTL); err != nil {
			return nil, errors.Wrap(err, "failed to scan row")
		}

		cluster.SnapshotSchedule = snapshotSchedule.String
		cluster.SnapshotTTL = snapshotTTL.String

		clusters = append(clusters, &cluster)
	}

	return clusters, nil
}

// GetClusterID retrieves the authoritative cluster_id from the database.
// Returns empty string if unavailable (no database access or no clusters).
// This should be used as a fallback when the kotsadm-id ConfigMap is unavailable.
func (s *KOTSStore) GetClusterID() string {
	clusters, err := s.ListClusters()
	if err != nil {
		logger.Debug("Failed to get database cluster ID", zap.Error(err))
		return ""
	}
	if len(clusters) == 0 {
		return ""
	}
	return clusters[0].ClusterID
}

func (s *KOTSStore) GetClusterIDFromSlug(slug string) (string, error) {
	db := persistence.MustGetDBSession()
	query := `select id from cluster where slug = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{slug},
	})
	if err != nil {
		return "", fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

	if !rows.Next() {
		return "", ErrNotFound
	}

	var clusterID string
	if err := rows.Scan(&clusterID); err != nil {
		return "", errors.Wrap(err, "failed to scan")
	}

	return clusterID, nil
}

func (s *KOTSStore) GetClusterIDFromDeployToken(deployToken string) (string, error) {
	db := persistence.MustGetDBSession()
	query := `select id from cluster where token = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{deployToken},
	})
	if err != nil {
		return "", fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

	if !rows.Next() {
		return "", ErrNotFound
	}

	var clusterID string
	if err := rows.Scan(&clusterID); err != nil {
		return "", errors.Wrap(err, "failed to scan")
	}

	return clusterID, nil
}

func (s *KOTSStore) CreateNewCluster(userID string, isAllUsers bool, title string, token string) (string, error) {
	clusterID := rand.StringWithCharset(32, rand.LOWER_CASE)
	clusterSlug := slug.Make(title)

	db := persistence.MustGetDBSession()

	foundUniqueSlug := false
	for i := 0; !foundUniqueSlug; i++ {
		slugProposal := clusterSlug
		if i > 0 {
			slugProposal = fmt.Sprintf("%s-%d", clusterSlug, i)
		}

		query := `select count(1) as count from cluster where slug = ?`
		rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: []interface{}{slugProposal},
		})
		if err != nil {
			return "", fmt.Errorf("failed to query: %v: %v", err, rows.Err)
		}
		if !rows.Next() {
			return "", ErrNotFound
		}

		var count int
		if err := rows.Scan(&count); err != nil {
			return "", errors.Wrap(err, "failed to scan")
		}

		if count == 0 {
			clusterSlug = slugProposal
			foundUniqueSlug = true
			break
		}
	}

	if token == "" {
		token = rand.StringWithCharset(32, rand.LOWER_CASE)
	}

	statements := []gorqlite.ParameterizedStatement{}
	statements = append(statements, gorqlite.ParameterizedStatement{
		Query:     `insert into cluster (id, title, slug, created_at, cluster_type, is_all_users, token) values (?, ?, ?, ?, ?, ?, ?)`,
		Arguments: []interface{}{clusterID, title, clusterSlug, time.Now().Unix(), "ship", isAllUsers, token},
	})

	if userID != "" {
		statements = append(statements, gorqlite.ParameterizedStatement{
			Query:     `insert into user_cluster (user_id, cluster_id) values (?, ?)`,
			Arguments: []interface{}{userID, clusterID},
		})
	}

	if wrs, err := db.WriteParameterized(statements); err != nil {
		wrErrs := []error{}
		for _, wr := range wrs {
			wrErrs = append(wrErrs, wr.Err)
		}
		return "", fmt.Errorf("failed to write: %v: %v", err, wrErrs)
	}

	return clusterID, nil
}

func (s *KOTSStore) SetInstanceSnapshotTTL(clusterID string, snapshotTTL string) error {
	logger.Debug("Setting instance snapshot TTL",
		zap.String("clusterID", clusterID))

	db := persistence.MustGetDBSession()
	query := `update cluster set snapshot_ttl = ? where id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{snapshotTTL, clusterID},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}

func (s *KOTSStore) SetInstanceSnapshotSchedule(clusterID string, snapshotSchedule string) error {
	logger.Debug("Setting instance snapshot Schedule",
		zap.String("clusterID", clusterID))

	db := persistence.MustGetDBSession()
	query := `update cluster set snapshot_schedule = ? where id = ?`
	wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{snapshotSchedule, clusterID},
	})
	if err != nil {
		return fmt.Errorf("failed to write: %v: %v", err, wr.Err)
	}

	return nil
}
