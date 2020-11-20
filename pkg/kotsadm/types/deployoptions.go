package types

import (
	"io"
	"time"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	identitytypes "github.com/replicatedhq/kots/pkg/identity/types"
	ingresstypes "github.com/replicatedhq/kots/pkg/ingress/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type DeployOptions struct {
	Namespace                 string
	KubernetesConfigFlags     *genericclioptions.ConfigFlags
	Context                   string
	SharedPassword            string
	SharedPasswordBcrypt      string
	S3AccessKey               string
	S3SecretKey               string
	JWT                       string
	PostgresPassword          string
	APIEncryptionKey          string
	AutoCreateClusterToken    string
	ServiceType               string
	NodePort                  int32
	ApplicationMetadata       []byte
	LimitRange                *corev1.LimitRange
	IsOpenShift               bool
	License                   *kotsv1beta1.License
	ConfigValues              *kotsv1beta1.ConfigValues
	Airgap                    bool
	AirgapRootDir             string
	AppImagesPushed           bool
	ProgressWriter            io.Writer
	StorageBaseURI            string
	StorageBaseURIPlainHTTP   bool
	IncludeMinio              bool
	IncludeDockerDistribution bool
	Timeout                   time.Duration
	HTTPProxyEnvValue         string
	HTTPSProxyEnvValue        string
	NoProxyEnvValue           string
	ExcludeAdminConsole       bool
	EnsureKotsadmConfig       bool

	IdentityConfig identitytypes.Config
	IngressConfig  ingresstypes.Config

	KotsadmOptions KotsadmOptions
}
