package s3pg

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
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
