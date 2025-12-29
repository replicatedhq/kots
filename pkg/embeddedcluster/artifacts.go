package embeddedcluster

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
	k8syaml "sigs.k8s.io/yaml"
)

// DistributeArtifacts distributes artifacts to all nodes in the cluster.
// This is called during app-only upgrades to ensure new nodes can join successfully.
func DistributeArtifacts(
	ctx context.Context,
	in *embeddedclusterv1beta1.Installation,
	registrySettings registrytypes.RegistrySettings,
	license *licensewrapper.LicenseWrapper,
	versionLabel string,
) error {
	var bin string

	if in.Spec.AirGap {
		artifact := in.Spec.Artifacts.AdditionalArtifacts["operator"]
		if artifact == "" {
			return fmt.Errorf("missing operator binary in airgap artifacts")
		}

		b, err := pullUpgradeBinaryFromRegistry(ctx, registrySettings, artifact)
		if err != nil {
			return fmt.Errorf("pull operator binary from registry: %w", err)
		}
		bin = b
	} else {
		b, err := downloadUpgradeBinary(ctx, license, versionLabel)
		if err != nil {
			return fmt.Errorf("download operator binary: %w", err)
		}
		bin = b
	}
	defer os.RemoveAll(bin)

	err := os.Chmod(bin, 0755)
	if err != nil {
		return fmt.Errorf("chmod operator binary: %w", err)
	}

	installationData, err := k8syaml.Marshal(in)
	if err != nil {
		return fmt.Errorf("marshal installation: %w", err)
	}

	args := []string{
		"distribute-artifacts",
		"--installation", "-",
		"--license-id", license.GetLicenseID(),
		"--app-slug", license.GetAppSlug(),
		"--channel-id", license.GetChannelID(),
		"--app-version", versionLabel,
	}

	log.Printf("Running distribute-artifacts command with args %q ...", maskLicenseIDInArgs(args))

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdin = strings.NewReader(string(installationData))

	// create pipe for capturing output
	pr, pw := io.Pipe()
	defer pw.Close()

	// capture stderr separately so we can return it in the error
	var stderr bytes.Buffer
	cmd.Stdout = pw
	cmd.Stderr = io.MultiWriter(pw, &stderr)

	// stream output to logs
	go func() {
		defer pr.Close()
		log.Println("Distribute artifacts command output:")
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			log.Println("  " + scanner.Text())
		}
	}()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run distribute-artifacts command: %w: %s", err, stderr.String())
	}

	return nil
}
