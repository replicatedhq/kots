package k8s

import (
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmapiv1beta2 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta2"
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

	bts, err := kubeadm.NewBootstrapTokenString(token)
	if err != nil {
		return "", errors.Wrap(err, "new kubeadm token string")
	}

	duration := &metav1.Duration{Duration: ttl}

	if err := tokenphase.UpdateOrCreateTokens(client, false, []kubeadm.BootstrapToken{
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
	client, err := Clientset()
	if err != nil {
		return "", errors.Wrap(err, "failed to create clientset")
	}

	config, err := kubeadmconfig.DefaultedInitConfiguration(&kubeadmapiv1beta2.InitConfiguration{}, &kubeadmapiv1beta2.ClusterConfiguration{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get kotsadm config")
	}

	key, err := copycerts.CreateCertificateKey()
	if err != nil {
		return "", errors.Wrap(err, "failed to genertae key")
	}
	config.CertificateKey = key

	err = copycerts.UploadCerts(client, config, key)
	if err != nil {
		return "", errors.Wrap(err, "failed to upload cert")
	}

	return key, nil
}
