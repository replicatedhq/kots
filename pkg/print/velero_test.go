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
// version is unknown and the flags must resolve to the newest behavior: node
// agent with the implicit (kopia) default uploader, i.e. no --uploader-type.
// These guard against re-introducing a hardcoded restic uploader literal.

func TestVeleroInstallationInstructionsForUI_usesNodeAgent(t *testing.T) {
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
			assertNodeAgentFlags(t, installCmd)
		})
	}
}

func TestVeleroInstallationInstructionsForCLI_usesNodeAgent(t *testing.T) {
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
			assertNodeAgentFlags(t, buf.String())
		})
	}
}

func assertNodeAgentFlags(t *testing.T, out string) {
	t.Helper()
	if !strings.Contains(out, "--use-node-agent") {
		t.Errorf("expected --use-node-agent in output: %s", out)
	}
	// kopia is the implicit default uploader, so no --uploader-type should be set.
	if strings.Contains(out, "--uploader-type") {
		t.Errorf("unexpected --uploader-type in output: %s", out)
	}
}
