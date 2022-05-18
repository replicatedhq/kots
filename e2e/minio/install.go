package minio

import (
	"fmt"
	"time"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/replicatedhq/kots/e2e/helm"
)

const (
	DefaultNamespace = "minio"
	DefaultService   = "minio"
	DefaultAccessKey = "accessKey"
	DefaultSecretKey = "secretKey"
	DefaultBucket    = "bucket1"
)

type Minio struct {
	namespace string
	service   string
	accessKey string
	secretKey string
	bucket    string
}

type Options struct {
	Namespace string
	Service   string
	AccessKey string
	SecretKey string
	Bucket    string
}

func New(opts Options) Minio {
	m := Minio{
		namespace: opts.Namespace,
		service:   opts.Service,
		accessKey: opts.AccessKey,
		secretKey: opts.SecretKey,
		bucket:    opts.Bucket,
	}
	if m.namespace == "" {
		m.namespace = DefaultNamespace
	}
	if m.service == "" {
		m.service = DefaultService
	}
	if m.accessKey == "" {
		m.accessKey = DefaultAccessKey
	}
	if m.secretKey == "" {
		m.secretKey = DefaultSecretKey
	}
	if m.bucket == "" {
		m.bucket = DefaultBucket
	}
	return m
}

func (m *Minio) Install(helmCLI *helm.CLI, kubeconfig string) {
	session, err := helmCLI.RepoAdd("minio", "https://charts.min.io/")
	Expect(err).WithOffset(1).Should(Succeed(), "helm repo add")
	Eventually(session).WithOffset(1).WithTimeout(time.Minute).Should(gexec.Exit(0), "helm repo add")

	session, err = helmCLI.Install(
		kubeconfig,
		"--create-namespace",
		fmt.Sprintf("--namespace=%s", m.namespace),
		"--wait",
		"--set=mode=standalone",
		"--set=replicas=1",
		"--set=resources.requests.memory=128Mi",
		"--set=persistence.enabled=false",
		"--set=rootUser=rootuser,rootPassword=rootpass123",
		fmt.Sprintf("--set=users[0].accessKey=%s,users[0].secretKey=%s,users[0].policy=readwrite", m.accessKey, m.secretKey),
		fmt.Sprintf("--set=buckets[0].name=%s,buckets[0].policy=none,buckets[0].purge=false", m.bucket),
		m.service,
		"minio/minio",
	)
	Expect(err).WithOffset(1).Should(Succeed(), "helm install")
	Eventually(session).WithOffset(1).WithTimeout(2*time.Minute).Should(gexec.Exit(0), "helm install")
}

func (m *Minio) GetURL() string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:9000", m.service, m.namespace)
}

func (m *Minio) GetAccessKey() string {
	return m.accessKey
}

func (m *Minio) GetSecretKey() string {
	return m.secretKey
}

func (m *Minio) GetBucket() string {
	return m.bucket
}
