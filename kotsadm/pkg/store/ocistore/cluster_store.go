package ocistore

import (
	"github.com/pkg/errors"
)

const (
	ClusterListConfigmapName = "kotsadm-clusters"
	ClusterDeployTokenSecret = "kotsadm-clustertokens"
)

func (s OCIStore) ListClusters() (map[string]string, error) {
	return nil, ErrNotImplemented
}

func (s OCIStore) GetClusterIDFromDeployToken(deployToken string) (string, error) {
	secret, err := s.getSecret(ClusterDeployTokenSecret)
	if err != nil {
		return "", errors.Wrap(err, "failed to get cluster deploy token secret")
	}

	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}

	clusterID, ok := secret.Data[deployToken]
	if !ok {
		return "", errors.New("cluster deploy token not found")
	}

	return string(clusterID), nil
}
