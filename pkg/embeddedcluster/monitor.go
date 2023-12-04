package embeddedcluster

import (
	"context"
	"fmt"
	"time"

	"github.com/replicatedhq/embedded-cluster-operator/api/v1beta1"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	statetypes "github.com/replicatedhq/kots/pkg/appstate/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"k8s.io/client-go/kubernetes"
)

// StartInstallationMonitor starts a goroutine that monitors the embedded cluster installation
// and starts the upgrade process if necessary.
func StartInstallationMonitor(ctx context.Context) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return fmt.Errorf("failed to get kubeclient: %w", err)
	}
	if isembedded, err := IsEmbeddedCluster(clientset); err != nil {
		return fmt.Errorf("failed to check if embedded: %w", err)
	} else if !isembedded {
		return nil
	}
	mon := monitor{
		store:     store.GetStore(),
		clientset: clientset,
	}
	go mon.start(ctx)
	return nil
}

// monitor is a struct that groups all methods needed to monitor the embedded cluster installation.
type monitor struct {
	store     store.Store
	clientset *kubernetes.Clientset
}

// getApp returns the app deployed on top of the embedded cluster.
func (m *monitor) getApp(ctx context.Context) (*apptypes.App, error) {
	apps, err := m.store.ListInstalledApps()
	if err != nil {
		return nil, fmt.Errorf("failed to list installed apps: %w", err)
	} else if len(apps) == 0 {
		return nil, nil
	}
	return apps[0], nil
}

// maybeStartUpgrade checks if the embedded cluster is in a state that requires an upgrade. If so,
// it starts the upgrade process. We only start an upgrade if the following conditions are met:
// - We have an app deployed on top of the embedded cluster.
// - The deployed app version is in ready state.
// - The app has an embedded cluster configuration.
// - The app embedded cluster configuration differs from the current embedded cluster config.
func (m *monitor) maybeStartUpgrade(ctx context.Context) error {
	app, err := m.getApp(ctx)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	} else if app == nil {
		return nil
	}
	cid, err := m.store.GetClusterIDFromSlug("this-cluster")
	if err != nil {
		return fmt.Errorf("failed to get cluster id: %w", err)
	}
	version, err := m.store.GetCurrentDownstreamVersion(app.ID, cid)
	if err != nil {
		return fmt.Errorf("failed to get downstream version: %w", err)
	}
	appv, err := m.store.GetAppVersion(app.ID, version.Sequence)
	if err != nil {
		return fmt.Errorf("failed to get app version: %w", err)
	}
	kinds := appv.KOTSKinds
	notDeployed := version.Status != storetypes.VersionDeployed
	noClusterConfig := kinds == nil || kinds.EmbeddedClusterConfig == nil
	if notDeployed || noClusterConfig {
		return nil
	}
	status, err := m.store.GetAppStatus(app.ID)
	if err != nil {
		return fmt.Errorf("failed to get app status: %w", err)
	}
	if statetypes.GetState(status.ResourceStates) != statetypes.StateReady {
		return nil
	}
	spec := kinds.EmbeddedClusterConfig.Spec
	if upgrade, err := RequiresUpgrade(ctx, spec); err != nil {
		return fmt.Errorf("failed to check if upgrade is required: %w", err)
	} else if !upgrade {
		return nil
	}
	if err := StartClusterUpgrade(ctx, spec); err != nil {
		return fmt.Errorf("failed to start cluster upgrade: %w", err)
	}
	return nil
}

// updateClusterState updates the cluster state in the database. Gets the state from the cluster
// by reading the latest embedded cluster installation CRD.
func (m *monitor) updateClusterState(ctx context.Context) error {
	installation, err := GetCurrentInstallation(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current installation: %w", err)
	}
	state := v1beta1.InstallationStateUnknown
	if installation.Status.State != "" {
		state = installation.Status.State
	}
	if err := m.store.SetEmbeddedClusterState(state); err != nil {
		return fmt.Errorf("failed to update embedded cluster state: %w", err)
	}
	return nil
}

// start starts the monitor loop. Only returns when the context is cancelled. We first update
// the cluster state and later maybe start an upgrade. We sleep for 5 seconds between each
// iteration.
func (m *monitor) start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 5):
		}
		if err := m.updateClusterState(ctx); err != nil {
			logger.Errorf("embeddedcluster monitor: fail updating state: %v", err)
		}
		if err := m.maybeStartUpgrade(ctx); err != nil {
			logger.Errorf("embeddedcluster monitor: upgrade failure: %v", err)
		}
	}
}
