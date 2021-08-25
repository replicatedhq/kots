package k8sutil

import (
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
	bootstraptokenv1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/bootstraptoken/v1"
	kubeadmapiv1beta3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	tokenphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/bootstraptoken/node"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/copycerts"
	kubeadmconfig "k8s.io/kubernetes/cmd/kubeadm/app/util/config"
)

// GenerateBootstrapToken will generate a node join token for kubeadm.
// ttl defines the time to live for this token.
func GenerateBootstrapToken(client kubernetes.Interface, ttl time.Duration) (string, error) {
	token, err := bootstraputil.GenerateBootstrapToken()
	if err != nil {
		return "", errors.Wrap(err, "generate kubeadm token")
	}

	bts, err := bootstraptokenv1.NewBootstrapTokenString(token)
	if err != nil {
		return "", errors.Wrap(err, "new kubeadm token string")
	}

	duration := &metav1.Duration{Duration: ttl}

	if err := tokenphase.UpdateOrCreateTokens(client, false, []bootstraptokenv1.BootstrapToken{
		{
			Token:  bts,
			TTL:    duration,
			Usages: []string{"authentication", "signing"},
			Groups: []string{kubeadmconstants.NodeBootstrapTokenAuthGroup},
		},
	}); err != nil {
		return "", errors.Wrap(err, "create kubeadm token")
	}

	return token, nil
}

func UploadCertsWithNewKey() (string, error) {
	client, err := GetClientset()
	if err != nil {
		return "", errors.Wrap(err, "failed to create clientset")
	}

	config, err := kubeadmconfig.DefaultedInitConfiguration(&kubeadmapiv1beta3.InitConfiguration{}, &kubeadmapiv1beta3.ClusterConfiguration{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get kotsadm config")
	}

	key, err := copycerts.CreateCertificateKey()
	if err != nil {
		return "", errors.Wrap(err, "failed to generate key")
	}
	config.CertificateKey = key

	err = copycerts.UploadCerts(client, config, key)
	if err != nil {
		return "", errors.Wrap(err, "failed to upload cert")
	}

	return key, nil
}
