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
	DefaultNamespace   = "minio"
	DefaultReleaseName = "minio"
	DefaultAccessKey   = "accessKey"
	DefaultSecretKey   = "secretKey"
	DefaultBucket      = "bucket1"
)

type Minio struct {
	options Options
}

type Options struct {
	Namespace   string
	ReleaseName string
	AccessKey   string
	SecretKey   string
	Bucket      string
}

func New(opts Options) Minio {
	m := Minio{
		Options{
			Namespace:   opts.Namespace,
			ReleaseName: opts.ReleaseName,
			AccessKey:   opts.AccessKey,
			SecretKey:   opts.SecretKey,
			Bucket:      opts.Bucket,
		},
	}
	if m.options.Namespace == "" {
		m.options.Namespace = DefaultNamespace
	}
	if m.options.ReleaseName == "" {
		m.options.ReleaseName = DefaultReleaseName
	}
	if m.options.AccessKey == "" {
		m.options.AccessKey = DefaultAccessKey
	}
	if m.options.SecretKey == "" {
		m.options.SecretKey = DefaultSecretKey
	}
	if m.options.Bucket == "" {
		m.options.Bucket = DefaultBucket
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
		fmt.Sprintf("--namespace=%s", m.options.Namespace),
		"--wait",
		"--set=mode=standalone",
		"--set=replicas=1",
		"--set=resources.requests.memory=128Mi",
		"--set=persistence.enabled=false",
		"--set=rootUser=rootuser,rootPassword=rootpass123",
		fmt.Sprintf("--set=users[0].accessKey=%s,users[0].secretKey=%s,users[0].policy=readwrite", m.GetAccessKey(), m.GetSecretKey()),
		fmt.Sprintf("--set=buckets[0].name=%s,buckets[0].policy=none,buckets[0].purge=false", m.GetBucket()),
		m.options.ReleaseName,
		"minio/minio",
		"--version=v5.0.13",
	)
	Expect(err).WithOffset(1).Should(Succeed(), "helm install")
	Eventually(session).WithOffset(1).WithTimeout(2*time.Minute).Should(gexec.Exit(0), "helm install")
}

func (m *Minio) GetURL() string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:9000", m.options.ReleaseName, m.options.Namespace)
}

func (m *Minio) GetAccessKey() string {
	return m.options.AccessKey
}

func (m *Minio) GetSecretKey() string {
	return m.options.SecretKey
}

func (m *Minio) GetBucket() string {
	return m.options.Bucket
}
