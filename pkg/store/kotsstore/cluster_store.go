package kotsstore

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/rand"
	"go.uber.org/zap"
)

func (s *KOTSStore) ListClusters() ([]*downstreamtypes.Downstream, error) {
	db := persistence.MustGetDBSession()

	query := `select id, slug, title, snapshot_schedule, snapshot_ttl from cluster` // TODO the current sequence
	rows, err := db.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query clusters")
	}
	defer rows.Close()

	clusters := []*downstreamtypes.Downstream{}
	for rows.Next() {
		cluster := downstreamtypes.Downstream{}

		var snapshotSchedule sql.NullString
		var snapshotTTL sql.NullString

		if err := rows.Scan(&cluster.ClusterID, &cluster.ClusterSlug, &cluster.Name, &snapshotSchedule, &snapshotTTL); err != nil {
			return nil, errors.Wrap(err, "failed to scan row")
		}

		cluster.SnapshotSchedule = snapshotSchedule.String
		cluster.SnapshotTTL = snapshotTTL.String

		clusters = append(clusters, &cluster)
	}

	return clusters, nil
}

func (s *KOTSStore) GetClusterIDFromSlug(slug string) (string, error) {
	db := persistence.MustGetDBSession()
	query := `select id from cluster where slug = $1`
	row := db.QueryRow(query, slug)

	var clusterID string
	if err := row.Scan(&clusterID); err != nil {
		return "", errors.Wrap(err, "failed to scan")
	}

	return clusterID, nil
}

func (s *KOTSStore) GetClusterIDFromDeployToken(deployToken string) (string, error) {
	db := persistence.MustGetDBSession()
	query := `select id from cluster where token = $1`
	row := db.QueryRow(query, deployToken)

	var clusterID string
	if err := row.Scan(&clusterID); err != nil {
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
		query := `select count(1) as count from cluster where slug = $1`
		row := db.QueryRow(query, slugProposal)

		var count int
		if err := row.Scan(&count); err != nil {
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

	tx, err := db.Begin()
	if err != nil {
		return "", errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	query := `insert into cluster (id, title, slug, created_at, updated_at, cluster_type, is_all_users, token) values ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err = db.Exec(query, clusterID, title, clusterSlug, time.Now(), nil, "ship", isAllUsers, token)
	if err != nil {
		return "", errors.Wrap(err, "failed to insert cluster row")
	}

	if userID != "" {
		query := `insert into user_cluster (user_id, cluster_id) values ($1, $2)`
		_, err := db.Exec(query, userID, clusterID)
		if err != nil {
			return "", errors.Wrap(err, "failed to insert user_cluster row")
		}
	}

	if err := tx.Commit(); err != nil {
		return "", errors.Wrap(err, "failed to commit transaction")
	}

	return clusterID, nil
}

func (s *KOTSStore) SetInstanceSnapshotTTL(clusterID string, snapshotTTL string) error {
	logger.Debug("Setting instance snapshot TTL",
		zap.String("clusterID", clusterID))
	db := persistence.MustGetDBSession()
	query := `update cluster set snapshot_ttl = $1 where id = $2`
	_, err := db.Exec(query, snapshotTTL, clusterID)
	if err != nil {
		return errors.Wrap(err, "failed to exec db query")
	}

	return nil
}

func (s *KOTSStore) SetInstanceSnapshotSchedule(clusterID string, snapshotSchedule string) error {
	logger.Debug("Setting instance snapshot Schedule",
		zap.String("clusterID", clusterID))
	db := persistence.MustGetDBSession()
	query := `update cluster set snapshot_schedule = $1 where id = $2`
	_, err := db.Exec(query, snapshotSchedule, clusterID)
	if err != nil {
		return errors.Wrap(err, "failed to exec db query")
	}

	return nil
}
