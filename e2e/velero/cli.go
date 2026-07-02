package velero

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/replicatedhq/kots/e2e/minio"
	"github.com/replicatedhq/kots/e2e/util"
	snapshottypes "github.com/replicatedhq/kots/pkg/snapshot/types"
)

type CLI struct {
	isOpenShift bool
}

func NewCLI(workspace string, isOpenShift bool) *CLI {
	return &CLI{
		isOpenShift: isOpenShift,
	}
}

func (v *CLI) Install(workspace, kubeconfig string, minio minio.Minio) {
	err := writeAWSCredentialsFile(workspace, minio.GetAccessKey(), minio.GetSecretKey())
	Expect(err).WithOffset(1).Should(Succeed(), "write aws-credentials file")

	session, err := v.install(workspace, kubeconfig, minio.GetURL(), minio.GetBucket())
	Expect(err).WithOffset(1).Should(Succeed(), "install")
	Eventually(session).WithOffset(1).WithTimeout(2*time.Minute).Should(gexec.Exit(0), "velero install")

	if v.isOpenShift {
		session, err = patchNodeAgentDaemonset(kubeconfig)
		Expect(err).WithOffset(1).Should(Succeed(), "patch node agent daemonset")
		Eventually(session).WithOffset(1).WithTimeout(2*time.Minute).Should(gexec.Exit(0), "kubectl patch")
	}
}

func (v *CLI) install(workspace, kubeconfig, s3Url, bucket string) (*gexec.Session, error) {
	// Get the velero CLI version to construct the fully-qualified image for OpenShift
	// compatibility and to select the version-correct file-system-backup flags.
	veleroVersion, err := getVeleroCLIVersion()
	if err != nil {
		return nil, fmt.Errorf("get velero version: %w", err)
	}
	veleroImage := fmt.Sprintf("docker.io/velero/velero:%s", veleroVersion)
	fsBackupFlags := snapshottypes.VeleroFSBackupFlags(veleroVersion)

	args := []string{
		"install",
		fmt.Sprintf("--kubeconfig=%s", kubeconfig),
		"--provider=aws",
		fmt.Sprintf("--image=%s", veleroImage),
		"--plugins=docker.io/velero/velero-plugin-for-aws:v1.14.2",
		fmt.Sprintf("--bucket=%s", bucket),
		fmt.Sprintf("--backup-location-config=region=minio,s3ForcePathStyle=true,s3Url=%s", s3Url),
		fmt.Sprintf("--secret-file=%s", filepath.Join(workspace, "aws-credentials")),
		fmt.Sprintf("--prefix=%s", "/smoke-test-velero"),
		"--use-volume-snapshots=false",
		"--velero-pod-cpu-request=250m",
		"--velero-pod-mem-request=128Mi",
		"--velero-pod-cpu-limit=500m",
		"--velero-pod-mem-limit=512Mi",
		"--node-agent-pod-cpu-request=250m",
		"--node-agent-pod-mem-request=256Mi",
		"--node-agent-pod-cpu-limit=500m",
		"--node-agent-pod-mem-limit=512Mi",
		"--wait",
	}
	args = append(args, fsBackupFlags...)
	return util.RunCommand(exec.Command("velero", args...))
}

// getVeleroCLIVersion returns the installed velero CLI version (e.g. "v1.16.2")
// by querying the CLI. The version drives both the fully-qualified image (for
// OpenShift CRI-O K8s 1.34+ compatibility) and the version-correct
// file-system-backup flags.
func getVeleroCLIVersion() (string, error) {
	cmd := exec.Command("velero", "version", "--client-only")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get velero version: %w (output: %s)", err, output)
	}

	// Parse version from output like "Client:\n  Version: v1.16.2\n..."
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Version:") {
			version := strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
			if version != "" {
				return version, nil
			}
		}
	}

	return "", fmt.Errorf("failed to parse velero version from output: %s", output)
}

func writeAWSCredentialsFile(workspace, accessKey, secretKey string) error {
	return ioutil.WriteFile(filepath.Join(workspace, "aws-credentials"), []byte(fmt.Sprintf(
		`[default]
aws_access_key_id=%s
aws_secret_access_key=%s
`,
		accessKey, secretKey,
	)), 0644)
}

func patchNodeAgentDaemonset(kubeconfig string) (*gexec.Session, error) {
	args := []string{
		fmt.Sprintf("--kubeconfig=%s", kubeconfig),
		"patch",
		"ds/node-agent",
		"--namespace=velero",
		"--type=json",
		"-p",
		`[{"op":"add","path":"/spec/template/spec/containers/0/securityContext","value": { "privileged": true}}]`,
	}
	return util.RunCommand(exec.Command("kubectl", args...))
}
