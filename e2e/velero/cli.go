package velero

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"time"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/replicatedhq/kots/e2e/minio"
	"github.com/replicatedhq/kots/e2e/util"
)

type CLI struct {
}

func NewCLI(workspace string) *CLI {
	return &CLI{}
}

func (v *CLI) Install(workspace, kubeconfig string, minio minio.Minio) {
	err := writeAWSCredentialsFile(workspace, minio.GetAccessKey(), minio.GetSecretKey())
	Expect(err).WithOffset(1).Should(Succeed(), "write aws-credentials file")

	session, err := v.install(workspace, kubeconfig, minio.GetURL(), minio.GetBucket())
	Expect(err).WithOffset(1).Should(Succeed(), "install")
	Eventually(session).WithOffset(1).WithTimeout(2*time.Minute).Should(gexec.Exit(0), "helm install")
}

func (v *CLI) install(workspace, kubeconfig, s3Url, bucket string) (*gexec.Session, error) {
	args := []string{
		"install",
		fmt.Sprintf("--kubeconfig=%s", kubeconfig),
		"--provider=aws",
		"--plugins=velero/velero-plugin-for-aws:v1.6.1",
		fmt.Sprintf("--bucket=%s", bucket),
		fmt.Sprintf("--backup-location-config=region=minio,s3ForcePathStyle=true,s3Url=%s", s3Url),
		fmt.Sprintf("--secret-file=%s", filepath.Join(workspace, "aws-credentials")),
		fmt.Sprintf("--prefix=%s", "/smoke-test-velero"),
		"--use-node-agent",
		"--uploader-type=restic",
		"--use-volume-snapshots=false",
		"--velero-pod-cpu-request=250m",
		"--velero-pod-mem-request=128Mi",
		"--velero-pod-cpu-limit=250m",
		"--velero-pod-mem-limit=128Mi",
		"--node-agent-pod-cpu-request=250m",
		"--node-agent-pod-mem-request=256Mi",
		"--node-agent-pod-cpu-limit=500m",
		"--node-agent-pod-mem-limit=512Mi",
		"--wait",
	}
	return util.RunCommand(exec.Command("velero", args...))
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
