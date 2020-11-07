package ocistore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gosimple/slug"
	"github.com/ocidb/ocidb/pkg/ocidb"
	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/kotsadm/pkg/downstream/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/rand"
	"go.uber.org/zap"
)

const (
	ClusterListConfigmapName = "kotsadm-clusters"
	ClusterDeployTokenSecret = "kotsadm-clustertokens"
)

func (s OCIStore) ListClusters() ([]*downstreamtypes.Downstream, error) {
	rows, err := s.connection.DB.Query("select id, slug, title, snapshot_schedule, snapshot_ttl from cluster")
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
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

func (s OCIStore) GetClusterIDFromSlug(slug string) (string, error) {
	query := `select id from cluster where slug = $1`
	row := s.connection.DB.QueryRow(query, slug)

	var clusterID string
	if err := row.Scan(&clusterID); err != nil {
		return "", errors.Wrap(err, "failed to scan")
	}

	return clusterID, nil
}

func (s OCIStore) GetClusterIDFromDeployToken(deployToken string) (string, error) {
	query := `select id from cluster where token = $1`
	row := s.connection.DB.QueryRow(query, deployToken)

	var clusterID string
	if err := row.Scan(&clusterID); err != nil {
		return "", errors.Wrap(err, "failed to scan")
	}

	return clusterID, nil
}

func (s OCIStore) CreateNewCluster(userID string, isAllUsers bool, title string, token string) (string, error) {
	logger.Debug("creating new cluster",
		zap.String("userID", userID),
		zap.Bool("isAllUsers", isAllUsers),
		zap.String("title", title))

	downstream := downstreamtypes.Downstream{
		ClusterID:   rand.StringWithCharset(32, rand.LOWER_CASE),
		ClusterSlug: slug.Make(title),
		Name:        title,
	}

	curentClusters, err := s.ListClusters()
	if err != nil {
		return "", errors.Wrap(err, "failed to list current clusters")
	}
	existingClusterSlugs := []string{}
	for _, currentCluster := range curentClusters {
		existingClusterSlugs = append(existingClusterSlugs, currentCluster.ClusterSlug)
	}

	foundUniqueSlug := false
	for i := 0; !foundUniqueSlug; i++ {
		slugProposal := downstream.ClusterSlug
		if i > 0 {
			slugProposal = fmt.Sprintf("%s-%d", downstream.ClusterSlug, i)
		}

		foundUniqueSlug = true
		for _, existingClusterSlug := range existingClusterSlugs {
			if slugProposal == existingClusterSlug {
				foundUniqueSlug = false
			}
		}

		if foundUniqueSlug {
			downstream.ClusterSlug = slugProposal
		}
	}

	if token == "" {
		token = rand.StringWithCharset(32, rand.LOWER_CASE)
	}

	tx, err := s.connection.DB.Begin()
	if err != nil {
		return "", errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	query := `insert into cluster (id, title, slug, created_at, updated_at, cluster_type, is_all_users, token) values ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err = s.connection.DB.Exec(query, downstream.ClusterID, title, downstream.ClusterSlug, time.Now(), nil, "ship", isAllUsers, token)
	if err != nil {
		return "", errors.Wrap(err, "failed to insert cluster row")
	}

	if userID != "" {
		query := `insert into user_cluster (user_id, cluster_id) values ($1, $2)`
		_, err := s.connection.DB.Exec(query, userID, downstream.ClusterID)
		if err != nil {
			return "", errors.Wrap(err, "failed to insert user_cluster row")
		}
	}

	if err := tx.Commit(); err != nil {
		return "", errors.Wrap(err, "failed to commit transaction")
	}

	if err := ocidb.Commit(context.TODO(), s.connection); err != nil {
		return "", errors.Wrap(err, "failed to commit")
	}

	return downstream.ClusterID, nil
}

func (s OCIStore) SetInstanceSnapshotTTL(clusterID string, snapshotTTL string) error {
	logger.Debug("Setting instance snapshot TTL",
		zap.String("clusterID", clusterID))

	query := `update cluster set snapshot_ttl = $1 where id = $2`
	_, err := s.connection.DB.Exec(query, snapshotTTL, clusterID)
	if err != nil {
		return errors.Wrap(err, "failed to exec db query")
	}
	if err := ocidb.Commit(context.TODO(), s.connection); err != nil {
		return errors.Wrap(err, "failed to commit")
	}

	return nil
}

func (s OCIStore) SetInstanceSnapshotSchedule(clusterID string, snapshotSchedule string) error {
	logger.Debug("Setting instance snapshot Schedule",
		zap.String("clusterID", clusterID))

	query := `update cluster set snapshot_schedule = $1 where id = $2`
	_, err := s.connection.DB.Exec(query, snapshotSchedule, clusterID)
	if err != nil {
		return errors.Wrap(err, "failed to exec db query")
	}
	if err := ocidb.Commit(context.TODO(), s.connection); err != nil {
		return errors.Wrap(err, "failed to commit")
	}

	return nil
}
