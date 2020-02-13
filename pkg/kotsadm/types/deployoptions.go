package types

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type DeployOptions struct {
	Namespace              string
	KubernetesConfigFlags  *genericclioptions.ConfigFlags
	Context                string
	IncludeShip            bool
	IncludeGitHub          bool
	SharedPassword         string
	SharedPasswordBcrypt   string
	S3AccessKey            string
	S3SecretKey            string
	JWT                    string
	PostgresPassword       string
	APIEncryptionKey       string
	AutoCreateClusterToken string
	ServiceType            string
	NodePort               int32
	Hostname               string
	ApplicationMetadata    []byte
	LimitRange             *corev1.LimitRange
	IsOpenShift            bool
}
