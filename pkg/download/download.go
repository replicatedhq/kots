package download

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type DownloadOptions struct {
	Namespace  string
	Kubeconfig string
}

func Download(path string, downloadOptions DownloadOptions) error {
	log := logger.NewLogger()
	log.ActionWithSpinner("Connecting to cluster")

	podName, err := findKotsadm(downloadOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "faioled to find kotsadm pod")
	}

	// set up port forwarding to get to it
	stopCh, err := k8sutil.PortForward(downloadOptions.Kubeconfig, 3000, 3000, downloadOptions.Namespace, podName)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to start port forwarding")
	}
	defer close(stopCh)

	resp, err := http.Get("http://localhost:3000/api/v1/kots")
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to get from kotsadm")
	}
	defer resp.Body.Close()

	tmpFile, err := ioutil.TempFile("", "kots")
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to create temp file")
	}
	defer os.Remove(tmpFile.Name())

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to write archive")
	}

	log.FinishSpinner()

	return nil
}

func findKotsadm(downloadOptions DownloadOptions) (string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "", errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to create kubernetes clientset")
	}

	pods, err := clientset.CoreV1().Pods(downloadOptions.Namespace).List(metav1.ListOptions{LabelSelector: "app=kotsadm-api"})
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
