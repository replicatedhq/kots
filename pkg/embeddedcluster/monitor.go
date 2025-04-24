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
	if kotsKinds == nil || kotsKinds.EmbeddedClusterConfig == nil {
		return false, nil
	}
	if !util.IsEmbeddedCluster() {
		return false, nil
	}
	curcfg, err := ClusterConfig(ctx, kbClient)
	if err != nil {
		// if there is no installation object we can't start an upgrade. this is a valid
		// scenario specially during cluster bootstrap. as we do not need to upgrade the
		// cluster just after its installation we can return nil here.
		// (the cluster in the first kots version will match the cluster installed during bootstrap)
		if errors.Is(err, ErrNoInstallations) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get current cluster config: %w", err)
	}
	serializedCur, err := json.Marshal(curcfg)
	if err != nil {
		return false, err
	}
	serializedNew, err := json.Marshal(kotsKinds.EmbeddedClusterConfig.Spec)
	if err != nil {
		return false, err
	}
	return !bytes.Equal(serializedCur, serializedNew), nil
}

func StartClusterUpgrade(ctx context.Context, kotsKinds *kotsutil.KotsKinds, channelSlug string, registrySettings registrytypes.RegistrySettings) error {
	spec := kotsKinds.EmbeddedClusterConfig.Spec
	artifacts := GetArtifactsFromInstallation(kotsKinds.Installation)

	if err := startClusterUpgrade(ctx, spec, artifacts, registrySettings, kotsKinds.License, channelSlug, kotsKinds.Installation.Spec.VersionLabel); err != nil {
		return fmt.Errorf("failed to start cluster upgrade: %w", err)
	}
	logger.Info("started cluster upgrade")

	return nil
}
