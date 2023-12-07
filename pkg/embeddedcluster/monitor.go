package embeddedcluster

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/replicatedhq/embedded-cluster-operator/api/v1beta1"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
)

var stateMut = sync.Mutex{}

// MaybeStartClusterUpgrade checks if the embedded cluster is in a state that requires an upgrade. If so,
// it starts the upgrade process. We only start an upgrade if the following conditions are met:
// - The app has an embedded cluster configuration.
// - The app embedded cluster configuration differs from the current embedded cluster config.
func MaybeStartClusterUpgrade(ctx context.Context, store store.Store, conf *v1beta1.Config) error {
	if conf == nil {
		return nil
	}

	spec := conf.Spec
	if upgrade, err := RequiresUpgrade(ctx, spec); err != nil {
		return fmt.Errorf("failed to check if upgrade is required: %w", err)
	} else if !upgrade {
		return nil
	}
	if err := StartClusterUpgrade(ctx, spec); err != nil {
		return fmt.Errorf("failed to start cluster upgrade: %w", err)
	}

	go watchClusterState(ctx, store)

	return nil
}

// watchClusterState checks the status of the installation object and updates the cluster state
// after the cluster state has been 'ready' for 5 minutes, it will exit the loop.
// this function is blocking and should be run in a goroutine.
// if it is called multiple times, only one instance will run.
// TODO: implement exit condition
func watchClusterState(ctx context.Context, store store.Store) {
	stateMut.Lock()
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 5):
		}
		if err := updateClusterState(ctx, store); err != nil {
			logger.Errorf("embeddedcluster monitor: fail updating state: %v", err)
		}
	}
}

// updateClusterState updates the cluster state in the database. Gets the state from the cluster
// by reading the latest embedded cluster installation CRD.
func updateClusterState(ctx context.Context, store store.Store) error {
	installation, err := GetCurrentInstallation(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current installation: %w", err)
	}
	state := v1beta1.InstallationStateUnknown
	if installation.Status.State != "" {
		state = installation.Status.State
	}
	if err := store.SetEmbeddedClusterState(state); err != nil {
		return fmt.Errorf("failed to update embedded cluster state: %w", err)
	}
	return nil
}
