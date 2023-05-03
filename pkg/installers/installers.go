package installers

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kurl"
	kurlv1beta1 "github.com/replicatedhq/kurlkinds/pkg/apis/cluster/v1beta1"
	"github.com/replicatedhq/kurlkinds/client/kurlclientset"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetDeployedInstaller() (*kurlv1beta1.Installer, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	cm, err := kurl.ReadConfigMap(clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kurl-config")
	}

	installerId := cm.Data["installer_id"]

	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	kurlClientset, err := kurlclientset.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kurl clientset")
	}

	deployedInstaller, err := kurlClientset.ClusterV1beta1().Installers(v1.NamespaceDefault).Get(context.TODO(), installerId, v1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get installer %s", installerId)
	}

	return deployedInstaller, nil
}
