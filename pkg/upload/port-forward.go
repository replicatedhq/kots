package upload

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func StartPortForward(namespace string, kubernetesConfigFlags *genericclioptions.ConfigFlags, stopCh <-chan struct{}, log *logger.CLILogger) (int, <-chan error, error) {
	clientset, err := k8sutil.GetClientset(kubernetesConfigFlags)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to get clientset")
	}

	podName, err := k8sutil.FindKotsadm(clientset, namespace)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to find kotsadm pod")
	}

	// set up port forwarding to get to it
	localPort, errChan, err := k8sutil.PortForward(kubernetesConfigFlags, 0, 3000, namespace, podName, false, stopCh, log)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to start port forwarding")
	}

	return localPort, errChan, nil
}
