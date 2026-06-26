package print

import (
	"bytes"
	"strings"
	"testing"

	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	snapshottypes "github.com/replicatedhq/kots/pkg/snapshot/types"
)

// Install instructions are emitted before Velero exists in the cluster, so the
// version is unknown and the flags must resolve to the newest behavior (kopia).
// These guard against re-introducing a hardcoded restic uploader literal.

func TestVeleroInstallationInstructionsForUI_usesKopia(t *testing.T) {
	cases := []struct {
		name           string
		registryConfig kotsadmtypes.RegistryConfig
	}{
		{"online", kotsadmtypes.RegistryConfig{}},
		{"airgap", kotsadmtypes.RegistryConfig{OverrideRegistry: "registry.example.com"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			instructions := VeleroInstallationInstructionsForUI(snapshottypes.VeleroAWSPlugin, &tc.registryConfig, "kots-configure-cmd")

			var installCmd string
			for _, in := range instructions {
				if in.Type == VeleroInstallationInstructionCommand && strings.Contains(in.Action, "velero install") {
					installCmd = in.Action
				}
			}
			if installCmd == "" {
				t.Fatalf("no velero install command found in instructions: %+v", instructions)
			}
			assertKopiaFlags(t, installCmd)
		})
	}
}

func TestVeleroInstallationInstructionsForCLI_usesKopia(t *testing.T) {
	cases := []struct {
		name           string
		registryConfig kotsadmtypes.RegistryConfig
	}{
		{"online", kotsadmtypes.RegistryConfig{}},
		{"airgap", kotsadmtypes.RegistryConfig{OverrideRegistry: "registry.example.com"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			log := logger.NewCLILogger(&buf)
			VeleroInstallationInstructionsForCLI(log, snapshottypes.VeleroAWSPlugin, &tc.registryConfig, "kots-configure-cmd")
			assertKopiaFlags(t, buf.String())
		})
	}
}

func assertKopiaFlags(t *testing.T, out string) {
	t.Helper()
	if !strings.Contains(out, "--use-node-agent") {
		t.Errorf("expected --use-node-agent in output: %s", out)
	}
	if !strings.Contains(out, "--uploader-type=kopia") {
		t.Errorf("expected --uploader-type=kopia in output: %s", out)
	}
	if strings.Contains(out, "--uploader-type=restic") {
		t.Errorf("unexpected --uploader-type=restic in output: %s", out)
	}
}
