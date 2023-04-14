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
	"k8s.io/utils/pointer"
)

var (
	ErrWaitForPodTimeout = errors.New("timeout waiting for pod")
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
			return ErrWaitForPodTimeout
		}
	}
}

func PodsHaveTheSameOwner(pods []corev1.Pod) bool {
	if len(pods) == 0 {
		return false
	}

	for _, pod := range pods {
		if len(pod.OwnerReferences) == 0 {
			return false
		}
	}

	owner := pods[0].OwnerReferences[0]

	for _, pod := range pods {
		if pod.OwnerReferences[0].APIVersion != owner.APIVersion {
			return false
		}
		if pod.OwnerReferences[0].Kind != owner.Kind {
			return false
		}
		if pod.OwnerReferences[0].Name != owner.Name {
			return false
		}
	}

	return true
}

func SecurePodContext(user int64, group int64, isStrict bool) *corev1.PodSecurityContext {
	var context corev1.PodSecurityContext

	if isStrict {
		context = corev1.PodSecurityContext{
			RunAsNonRoot:       pointer.Bool(true),
			RunAsUser:          &user,
			RunAsGroup:         &group,
			FSGroup:            &group,
			SupplementalGroups: []int64{group},
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		}
	} else {
		context = corev1.PodSecurityContext{
			RunAsUser: pointer.Int64(user),
			FSGroup:   pointer.Int64(group),
		}
	}

	return &context
}
func SecureContainerContext(isStrict bool) *corev1.SecurityContext {
	var context *corev1.SecurityContext

	if isStrict {
		context = &corev1.SecurityContext{
			Privileged:               pointer.Bool(false),
			AllowPrivilegeEscalation: pointer.Bool(false),
			ReadOnlyRootFilesystem:   pointer.Bool(true),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{
					"ALL",
				},
			},
		}
	}
	return context
}
