package upload

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func StartPortForward(namespace string, kubeconfig string, stopCh <-chan struct{}, log *logger.Logger) (<-chan error, error) {
	podName, err := findKotsadm(namespace)
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

func findKotsadm(namespace string) (string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "", errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to create kubernetes clientset")
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: "app=kotsadm-api"})
	if err != nil {
		return "", errors.Wrap(err, "failed to list pods")
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			return pod.Name, nil
		}
	}

	return "", errors.New("unable to find kotsadm pod")
}
