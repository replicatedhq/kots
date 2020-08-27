package apiserver

import (
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
)

func bootstrap() error {
	if err := store.GetStore().Init(); err != nil {
		return errors.Wrap(err, "failed to init store")
	}

	if err := bootstrapClusterToken(); err != nil {
		return errors.Wrap(err, "failed to bootstrap cluster token")
	}

	return nil
}

func bootstrapClusterToken() error {
	if os.Getenv("AUTO_CREATE_CLUSTER_TOKEN") == "" {
		return errors.New("AUTO_CREATE_CLUSTER_TOKEN is not set")
	}

	_, err := store.GetStore().GetClusterIDFromDeployToken(os.Getenv("AUTO_CREATE_CLUSTER_TOKEN"))
	if err == nil {
		return nil
	}

	if err != nil && !store.GetStore().IsNotFound(err) {
		return errors.Wrap(err, "failed to lookup cluster ID")
	}

	_, err = store.GetStore().CreateNewCluster("", true, "this-cluster", os.Getenv("AUTO_CREATE_CLUSTER_TOKEN"))
	if err != nil {
		return errors.Wrap(err, "failed to create cluster")
	}

	return nil
}
