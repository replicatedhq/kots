package k8sutil

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetPodLogs(ctx context.Context, clientset kubernetes.Interface, pod *corev1.Pod, follow bool, maxLines *int64) ([]byte, error) {
	defaultMaxLines := int64(10000)

	podLogOpts := corev1.PodLogOptions{
		Container: pod.Spec.Containers[0].Name,
		Follow:    follow,
		TailLines: &defaultMaxLines,
	}

	if maxLines != nil {
		podLogOpts.TailLines = maxLines
	}

	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get log stream")
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	errChan := make(chan error, 0)
	go func() {
		_, err := io.Copy(buf, podLogs)
		errChan <- err
	}()

	select {
	case resErr := <-errChan:
		if resErr != nil {
			return nil, errors.Wrap(resErr, "failed to copy logs")
		} else {
			return buf.Bytes(), nil
		}
	case <-ctx.Done():
		return nil, errors.Wrap(ctx.Err(), "context ended copying logs")
	}
}

func WaitForPod(ctx context.Context, clientset kubernetes.Interface, namespace string, podName string, timeoutWaitingForPod time.Duration) error {
	start := time.Now()

	for {
		pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to get pod")
		}

		if pod.Status.Phase == corev1.PodRunning ||
			pod.Status.Phase == corev1.PodFailed ||
			pod.Status.Phase == corev1.PodSucceeded {
			return nil
		}
		if pod.Status.Phase == corev1.PodPending {
			for _, v := range pod.Status.ContainerStatuses {
				if v.State.Waiting != nil && v.State.Waiting.Reason == "ImagePullBackOff" {
					return errors.New("wait for pod aborted after getting pod status 'ImagePullBackOff'")
				}
			}
		}

		time.Sleep(time.Second)

		if time.Now().Sub(start) > timeoutWaitingForPod {
			return errors.New("timeout waiting for pod")
		}
	}
}
