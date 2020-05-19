package types

import (
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type DeployOptions struct {
	Namespace              string
	KubernetesConfigFlags  *genericclioptions.ConfigFlags
	Context                string
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
	ApplicationMetadata    []byte
	LimitRange             *corev1.LimitRange
	IsOpenShift            bool
	License                *kotsv1beta1.License
	UnsignedLicense        *kotsv1beta1.UnsignedLicense
	ConfigValues           *kotsv1beta1.ConfigValues
}
