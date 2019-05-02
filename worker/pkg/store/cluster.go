package store

import (
	"context"

	"github.com/pkg/errors"

	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
)

func (s *SQLStore) GetCluster(ctx context.Context, clusterID string) (*types.Cluster, error) {
	query := `select id, cluster_type, created_at from cluster where id = $1`
	row := s.db.QueryRowContext(ctx, query, clusterID)

	cluster := types.Cluster{}

	if err := row.Scan(&cluster.ID, &cluster.Type, &cluster.CreatedAt); err != nil {
		return nil, errors.Wrap(err, "get cluster")
	}

	if cluster.Type == "gitops" {
		query = `select owner, repo, branch, installation_id from cluster_github where cluster_id = $1`
		row = s.db.QueryRowContext(ctx, query, clusterID)

		if err := row.Scan(&cluster.GitHubOwner, &cluster.GitHubRepo, &cluster.GitHubBranch, &cluster.GitHubInstallationID); err != nil {
			return nil, errors.Wrap(err, "read cluster github")
		}
	}

	return &cluster, nil
}

func (s SQLStore) GetClusterForWatch(ctx context.Context, watchID string) (*types.Cluster, error) {
	query := `select cluster_id from watch_cluster where watch_id = $1`
	rows, err := s.db.QueryContext(ctx, query, watchID)
	if err != nil {
		return nil, errors.Wrap(err, "get clusterid from watchid")
	}

	if !rows.Next() {
		// No clusters
		return nil, nil
	}

	var clusterID string
	if err := rows.Scan(&clusterID); err != nil {
		return nil, errors.Wrap(err, "read clusterID")
	}

	return s.GetCluster(ctx, clusterID)
}

func (s SQLStore) GetGitHubPathForClusterWatch(ctx context.Context, clusterID string, watchID string) (string, error) {
	query := `select github_path from watch_cluster where watch_id = $1 and cluster_id = $2`;
	row := s.db.QueryRowContext(ctx, query, watchID, clusterID)

	githubPath := ""
	if err := row.Scan(&githubPath); err != nil {
		return "", errors.Wrap(err, "get github path")
	}

	return githubPath, nil
}
