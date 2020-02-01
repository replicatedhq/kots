package types

import (
	corev1 "k8s.io/api/core/v1"
)

type DeployOptions struct {
	Namespace              string
	Kubeconfig             string
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
	IsOpenShift            bool // true if the application is being deployed to an OpenShift cluster
	HostNetwork            bool // host-network flag was passed on CLI or in kURL
}
