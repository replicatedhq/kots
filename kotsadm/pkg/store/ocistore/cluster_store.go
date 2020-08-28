package ocistore

import (
	"encoding/json"
	"fmt"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/kotsadm/pkg/downstream/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/rand"
)

const (
	ClusterListConfigmapName = "kotsadm-clusters"
	ClusterDeployTokenSecret = "kotsadm-clustertokens"
)

func (s OCIStore) ListClusters() ([]*downstreamtypes.Downstream, error) {
	configMap, err := s.getConfigmap(ClusterListConfigmapName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clusters config map")
	}

	if configMap.Data == nil {
		configMap.Data = map[string]string{}
	}

	clusters := []*downstreamtypes.Downstream{}
	for _, data := range configMap.Data {
		cluster := downstreamtypes.Downstream{}
		if err := json.Unmarshal([]byte(data), &cluster); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal cluster")
		}

		clusters = append(clusters, &cluster)
	}

	return clusters, nil
}

func (s OCIStore) GetClusterIDFromSlug(slug string) (string, error) {
	return "", ErrNotImplemented
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

func (s OCIStore) CreateNewCluster(userID string, isAllUsers bool, title string, token string) (string, error) {
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

		foundUniqueSlug := true
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

	b, err := json.Marshal(downstream)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal cluster")
	}

	configMap, err := s.getConfigmap(ClusterListConfigmapName)
	if err != nil {
		return "", errors.Wrap(err, "failed to list clusters")
	}

	if configMap.Data == nil {
		configMap.Data = map[string]string{}
	}

	configMap.Data[downstream.ClusterID] = string(b)

	if err := s.updateConfigmap(configMap); err != nil {
		return "", errors.Wrap(err, "failed to update config map")
	}

	return downstream.ClusterID, nil
}
