package download

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/mholt/archiver"
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
	Overwrite  bool
}

func Download(appSlug string, path string, downloadOptions DownloadOptions) error {
	log := logger.NewLogger()
	log.ActionWithSpinner("Connecting to cluster")

	podName, err := findKotsadm(downloadOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to find kotsadm pod")
	}

	// set up port forwarding to get to it
	stopCh, errChan, err := k8sutil.PortForward(downloadOptions.Kubeconfig, 3000, 3000, downloadOptions.Namespace, podName, false)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to start port forwarding")
	}
	defer close(stopCh)

	go func() {
		select {
		case err := <-errChan:
			if err != nil {
				log.Error(err)
			}
		case <-stopCh:
		}
	}()

	resp, err := http.Get(fmt.Sprintf("http://localhost:3000/api/v1/kots/%s", appSlug))
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to get from kotsadm")
	}
	defer resp.Body.Close()

	tmpFile, err := ioutil.TempFile("", "kots")
	if err != nil {
		log.FinishSpinner()
		return errors.Wrap(err, "failed to create temp file")
	}
	defer os.Remove(tmpFile.Name())

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		log.FinishSpinner()
		return errors.Wrap(err, "failed to write archive")
	}

	// Delete the destination, if needed and requested
	if _, err := os.Stat(path); err == nil {
		if downloadOptions.Overwrite {
			if err := os.RemoveAll(path); err != nil {
				return errors.Wrap(err, "failed to delete existing download")
			}
		} else {
			log.FinishSpinner()
			log.ActionWithoutSpinner("")
			log.Info("Directory %s already exists. You can re-run this command with --overwrite to automatically overwrite it", path)
			log.ActionWithoutSpinner("")
			return errors.Errorf("directory already exists at %s", path)
		}
	}

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Unarchive(tmpFile.Name(), path); err != nil {
		return errors.Wrap(err, "failed to extract tar gz")
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
