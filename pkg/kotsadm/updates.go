package kotsadm

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/rand"
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type UpdateStatus string

const (
	UpdateRunning    UpdateStatus = "running"
	UpdateNotFound   UpdateStatus = "not-found"
	UpdateFailed     UpdateStatus = "failed"
	UpdateSuccessful UpdateStatus = "successful"
	UpdateUnknown    UpdateStatus = "unknown"
)

func GetUpdateUpdateStatus() (UpdateStatus, error) {
	pod, err := findUpdatePod()
	if err != nil {
		return UpdateUnknown, errors.Wrap(err, "failed to find update pod")
	}

	if pod == nil {
		return UpdateNotFound, nil
	}

	if pod.Status.Phase == corev1.PodSucceeded {
		return UpdateSuccessful, nil
	}

	if len(pod.Status.ContainerStatuses) == 0 {
		return UpdateUnknown, nil
	}

	cs := pod.Status.ContainerStatuses[0]

	if cs.State.Terminated == nil {
		if pod.CreationTimestamp.Add(5 * time.Minute).Before(time.Now()) {
			return UpdateNotFound, nil
		}

		return UpdateRunning, nil
	}

	if cs.State.Terminated.ExitCode != 0 {
		return UpdateFailed, nil
	}

	return UpdateSuccessful, nil
}

func UpdateToVersion(newVersion string) error {
	status, err := GetUpdateUpdateStatus()
	if err != nil {
		return errors.Wrap(err, "failed to check update status")
	}

	logger.Debugf("Current Admin Console update status is %s", status)

	if status == UpdateRunning {
		return errors.New("update already in progress")
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client")
	}

	ns := util.PodNamespace

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("kotsadm-updater-%s", rand.StringWithCharset(10, rand.LOWER_CASE)),
			Namespace: ns,
			Labels: map[string]string{
				"app":             "kotsadm-updater",
				"current-version": buildversion.Version(),
				"target-version":  newVersion,
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "kotsadm",
			RestartPolicy:      corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:    "kotsadm-updater",
					Image:   fmt.Sprintf("kotsadm/kotsadm:%s", newVersion),
					Command: []string{"/kots"},
					Args: []string{
						"admin-console",
						"upgrade",
						"-n",
						ns,
					},
				},
			},
		},
	}

	_, err = clientset.CoreV1().Pods(ns).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create update pod")
	}

	return nil
}

func findUpdatePod() (*corev1.Pod, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create k8s client")
	}

	selectorLabels := map[string]string{
		"app": "kotsadm-updater",
	}

	ns := util.PodNamespace

	pods, err := clientset.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list update pods")
	}

	var pod *corev1.Pod
	for _, p := range pods.Items {
		if pod == nil {
			pod = p.DeepCopy()
			continue
		}
		if pod.CreationTimestamp.Before(&p.CreationTimestamp) {
			pod = p.DeepCopy()
		}
	}

	return pod, nil
}
