package embeddedcluster

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/replicatedhq/embedded-cluster-operator/api/v1beta1"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	"k8s.io/client-go/kubernetes"
)

var stateMut = sync.Mutex{}

// MaybeStartClusterUpgrade checks if the embedded cluster is in a state that requires an upgrade. If so,
// it starts the upgrade process. We only start an upgrade if the following conditions are met:
// - The app has an embedded cluster configuration.
// - The app embedded cluster configuration differs from the current embedded cluster config.
// - The current cluster config (as part of the Installation object) already exists in the cluster.
func MaybeStartClusterUpgrade(ctx context.Context, client kubernetes.Interface, store store.Store, conf *v1beta1.Config) error {
	if conf == nil {
		return nil
	}

	if !util.IsEmbeddedCluster() {
		return nil
	}

	spec := conf.Spec
	if upgrade, err := RequiresUpgrade(ctx, spec); err != nil {
		// if there is no installation object we can't start an upgrade. this is a valid
		// scenario specially during cluster bootstrap. as we do not need to upgrade the
		// cluster just after its installation we can return nil here.
		// (the cluster in the first kots version will match the cluster installed during bootstrap)
		if errors.Is(err, ErrNoInstallations) {
			return nil
		}
		return fmt.Errorf("failed to check if upgrade is required: %w", err)
	} else if !upgrade {
		return nil
	}
	if err := startClusterUpgrade(ctx, spec); err != nil {
		return fmt.Errorf("failed to start cluster upgrade: %w", err)
	}

	go watchClusterState(ctx, store)

	return nil
}

// InitClusterState initializes the cluster state in the database. This should be called when the
// server launches.
func InitClusterState(ctx context.Context, client kubernetes.Interface, store store.Store) error {
	if util.IsEmbeddedCluster() {
		go watchClusterState(ctx, store)
		return nil
	}
	return nil
}

// watchClusterState checks the status of the installation object and updates the cluster state
// after the cluster state has been 'installed' for 30 seconds, it will exit the loop.
// this function is blocking and should be run in a goroutine.
// if it is called multiple times, only one instance will run.
func watchClusterState(ctx context.Context, store store.Store) {
	stateMut.Lock()
	defer stateMut.Unlock()
	numReady := 0
	lastState := ""
	for numReady < 6 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 5):
		}
		state, err := updateClusterState(ctx, store, lastState)
		if err != nil {
			logger.Errorf("embeddedcluster monitor: fail updating state: %v", err)
		}

		if state == v1beta1.InstallationStateInstalled {
			numReady++
		} else {
			numReady = 0
		}
		lastState = state
	}
}

// updateClusterState updates the cluster state in the database. Gets the state from the cluster
// by reading the latest embedded cluster installation CRD.
// If the lastState is the same as the current state, it will not update the database.
func updateClusterState(ctx context.Context, store store.Store, lastState string) (string, error) {
	installation, err := GetCurrentInstallation(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get current installation: %w", err)
	}
	state := v1beta1.InstallationStateUnknown
	if installation.Status.State != "" {
		state = installation.Status.State
	}
	// only update the state if it has changed
	if state != lastState {
		if err := store.SetEmbeddedClusterState(state); err != nil {
			return "", fmt.Errorf("failed to update embedded cluster state: %w", err)
		}
	}
	return state, nil
}
