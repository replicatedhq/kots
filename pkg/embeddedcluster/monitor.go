package embeddedcluster

import (
	"context"
	"fmt"
	"sync"

	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
)

var stateMut = sync.Mutex{}

// MaybeStartClusterUpgrade checks if the embedded cluster is in a state that requires an upgrade. If so,
// it starts the upgrade process. We only start an upgrade if the following conditions are met:
// - The app has an embedded cluster configuration.
// - The app embedded cluster configuration differs from the current embedded cluster config.
// - The current cluster config (as part of the Installation object) already exists in the cluster.
// Returns the name of the installation object if an upgrade was started.
func MaybeStartClusterUpgrade(ctx context.Context, kotsKinds *kotsutil.KotsKinds) error {
	if kotsKinds == nil || kotsKinds.EmbeddedClusterConfig == nil {
		return nil
	}

	if !util.IsEmbeddedCluster() {
		return nil
	}

	kbClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get kubeclient: %w", err)
	}

	spec := kotsKinds.EmbeddedClusterConfig.Spec
	if upgrade, err := RequiresUpgrade(ctx, kbClient, spec); err != nil {
		return fmt.Errorf("failed to check if upgrade is required: %w", err)
	} else if !upgrade {
		return nil
	}

	artifacts := getArtifactsFromInstallation(kotsKinds.Installation)

	if err := startClusterUpgrade(ctx, spec, artifacts, *kotsKinds.License); err != nil {
		return fmt.Errorf("failed to start cluster upgrade: %w", err)
	}
	logger.Info("started cluster upgrade")

	return nil
}
