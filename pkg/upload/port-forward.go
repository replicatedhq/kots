package upload

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
)

func StartPortForward(namespace string, stopCh <-chan struct{}, log *logger.CLILogger) (int, <-chan error, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to get clientset")
	}

	podName, err := k8sutil.FindKotsadm(clientset, namespace)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to find kotsadm pod")
	}

	// set up port forwarding to get to it
	localPort, errChan, err := k8sutil.PortForward(0, 3000, namespace, podName, false, stopCh, log)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to start port forwarding")
	}

	return localPort, errChan, nil
}
