package upload

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
)

func StartPortForward(namespace string, kubeconfig string, stopCh <-chan struct{}, log *logger.Logger) (<-chan error, error) {
	podName, err := k8sutil.FindKotsadm(namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find kotsadm pod")
	}

	// set up port forwarding to get to it
	errChan, err := k8sutil.PortForward(kubeconfig, 3000, 3000, namespace, podName, false, stopCh, log)
	if err != nil {
		return nil, errors.Wrap(err, "failed to start port forwarding")
	}

	return errChan, nil
}
