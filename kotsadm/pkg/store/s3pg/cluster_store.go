package s3pg

import (
	"fmt"
	"time"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	"github.com/replicatedhq/kots/kotsadm/pkg/rand"
)

func (s S3PGStore) ListClusters() (map[string]string, error) {
	db := persistence.MustGetPGSession()

	query := `select id, title from cluster`
	rows, err := db.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query clusters")
	}
	defer rows.Close()
	clusterIDs := map[string]string{}
	for rows.Next() {
		clusterID := ""
		name := ""
		if err := rows.Scan(&clusterID, &name); err != nil {
			return nil, errors.Wrap(err, "failed to scan row")
		}
		clusterIDs[clusterID] = name
	}

	return clusterIDs, nil
}

func (s S3PGStore) GetClusterIDFromSlug(slug string) (string, error) {
	db := persistence.MustGetPGSession()
	query := `select id from cluster where slug = $1`
	row := db.QueryRow(query, slug)

	var clusterID string
	if err := row.Scan(&clusterID); err != nil {
		return "", errors.Wrap(err, "failed to scan")
	}

	return clusterID, nil
}

func (s S3PGStore) GetClusterIDFromDeployToken(deployToken string) (string, error) {
	db := persistence.MustGetPGSession()
	query := `select id from cluster where token = $1`
	row := db.QueryRow(query, deployToken)

	var clusterID string
	if err := row.Scan(&clusterID); err != nil {
		return "", errors.Wrap(err, "failed to scan")
	}

	return clusterID, nil
}

func (s S3PGStore) LookupClusterID(clusterType string, title string, token string) (string, error) {
	db := persistence.MustGetPGSession()
	query := `select id from cluster where cluster_type = $1 and title = $2 and token = $3`
	row := db.QueryRow(query, clusterType, title, token)

	var clusterID string
	if err := row.Scan(&clusterID); err != nil {
		return "", errors.Wrap(err, "failed to scan")
	}

	return clusterID, nil
}

func (s S3PGStore) CreateNewCluster(userID string, isAllUsers bool, title string, token string) (string, error) {
	clusterID := rand.StringWithCharset(32, rand.LOWER_CASE)
	clusterSlug := slug.Make(title)

	db := persistence.MustGetPGSession()

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
