package embeddedcluster

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/util"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// RequiresClusterUpgrade returns true if the embedded cluster is in a state that requires an upgrade.
// This is determined by checking that:
// - The app has an embedded cluster configuration.
// - The app embedded cluster configuration differs from the current embedded cluster configuration.
// - The current cluster config (as part of the Installation object) already exists in the cluster.
func RequiresClusterUpgrade(ctx context.Context, kbClient kbclient.Client, kotsKinds *kotsutil.KotsKinds) (bool, error) {
	logger.Debugf("[Channel Switch Debug] RequiresClusterUpgrade called - kotsKinds nil: %t, EmbeddedClusterConfig nil: %t",
		kotsKinds == nil,
		kotsKinds != nil && kotsKinds.EmbeddedClusterConfig == nil)

	if kotsKinds == nil || kotsKinds.EmbeddedClusterConfig == nil {
		logger.Debugf("[Channel Switch Debug] RequiresClusterUpgrade returning false - no EmbeddedClusterConfig present")
		return false, nil
	}

	logger.Debugf("[Channel Switch Debug] RequiresClusterUpgrade - New EC config version: %s", kotsKinds.EmbeddedClusterConfig.Spec.Version)

	if !util.IsEmbeddedCluster() {
		logger.Debugf("[Channel Switch Debug] RequiresClusterUpgrade returning false - not an embedded cluster")
		return false, nil
	}

	curcfg, err := ClusterConfig(ctx, kbClient)
	if err != nil {
		// if there is no installation object we can't start an upgrade. this is a valid
		// scenario specially during cluster bootstrap. as we do not need to upgrade the
		// cluster just after its installation we can return nil here.
		// (the cluster in the first kots version will match the cluster installed during bootstrap)
		if errors.Is(err, ErrNoInstallations) {
			logger.Debugf("[Channel Switch Debug] RequiresClusterUpgrade returning false - no Installation CR found")
			return false, nil
		}
		return false, fmt.Errorf("failed to get current cluster config: %w", err)
	}

	logger.Debugf("[Channel Switch Debug] RequiresClusterUpgrade - Current cluster EC version: %s", curcfg.Version)

	serializedCur, err := json.Marshal(curcfg)
	if err != nil {
		return false, err
	}
	serializedNew, err := json.Marshal(kotsKinds.EmbeddedClusterConfig.Spec)
	if err != nil {
		return false, err
	}

	configsMatch := bytes.Equal(serializedCur, serializedNew)
	upgradeRequired := !configsMatch

	logger.Debugf("[Channel Switch Debug] RequiresClusterUpgrade - Configs match: %t, Upgrade required: %t", configsMatch, upgradeRequired)

	return upgradeRequired, nil
}

func StartClusterUpgrade(ctx context.Context, kotsKinds *kotsutil.KotsKinds, registrySettings registrytypes.RegistrySettings) error {
	spec := kotsKinds.EmbeddedClusterConfig.Spec
	artifacts := GetArtifactsFromInstallation(kotsKinds.Installation)

	if err := startClusterUpgrade(ctx, spec, artifacts, registrySettings, kotsKinds.License, kotsKinds.Installation.Spec.VersionLabel); err != nil {
		return fmt.Errorf("failed to start cluster upgrade: %w", err)
	}
	logger.Info("started cluster upgrade")

	return nil
}
