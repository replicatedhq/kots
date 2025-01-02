package kotsadm

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/rand"
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type UpdateStatus string

const (
	UpdateRunning    UpdateStatus = "running"
	UpdateNotFound   UpdateStatus = "not-found"
	UpdateFailed     UpdateStatus = "failed"
	UpdateSuccessful UpdateStatus = "successful"
	UpdateUnknown    UpdateStatus = "unknown"
)

func GetKotsUpdateStatus() (UpdateStatus, string, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return UpdateNotFound, "", errors.Wrap(err, "failed to create k8s client")
	}

	ctx := context.TODO()

	pod, err := findUpdatePod(ctx, clientset)
	if err != nil {
		return UpdateUnknown, "", errors.Wrap(err, "failed to find update pod")
	}

	if pod == nil {
		return UpdateNotFound, "", nil
	}

	if pod.CreationTimestamp.Add(5 * time.Minute).Before(time.Now()) {
		return UpdateNotFound, "", nil
	}

	lastLine, err := getLastLogLineFromPod(ctx, clientset, pod)
	if err != nil {
		logger.Debugf("failed to get last log line from pod: %v", err)
	}

	if len(pod.Status.ContainerStatuses) == 0 {
		return UpdateUnknown, lastLine, nil
	}

	cs := pod.Status.ContainerStatuses[0]

	if cs.State.Terminated == nil {
		if pod.CreationTimestamp.Add(5 * time.Minute).Before(time.Now()) {
			return UpdateNotFound, "", nil
		}

		return UpdateRunning, lastLine, nil
	}

	if cs.State.Terminated.ExitCode != 0 {
		return UpdateFailed, lastLine, nil
	}

	return UpdateSuccessful, lastLine, nil
}

func UpdateToVersion(newVersion string) error {
	status, _, err := GetKotsUpdateStatus()
	if err != nil {
		return errors.Wrap(err, "failed to check update status")
	}

	logger.Debugf("Current Admin Console update status is %s", status)

	if status == UpdateRunning {
		return errors.New("update already in progress")
	}

	ns := util.PodNamespace

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client")
	}

	registryConfig, err := GetRegistryConfigFromCluster(ns, clientset)
	if err != nil {
		return errors.Wrap(err, "failed to get kots options from cluster")
	}

	installationParams, err := kotsutil.GetInstallationParams(kotsadmtypes.KotsadmConfigMap)
	if err != nil {
		return errors.Wrap(err, "failed to get installation params")
	}

	args := []string{
		fmt.Sprintf("namespace=%s", ns),
		fmt.Sprintf("ensure-rbac=%v", installationParams.EnsureRBAC),
		fmt.Sprintf("skip-rbac-check=%v", installationParams.SkipRBACCheck),
		fmt.Sprintf("strict-security-context=%v", installationParams.StrictSecurityContext),
		fmt.Sprintf("wait-duration=%v", installationParams.WaitDuration),
		fmt.Sprintf("with-minio=%v", installationParams.WithMinio),
	}

	if registryConfig.OverrideRegistry != "" && !registryConfig.IsReadOnly {
		var registryValue string
		if registryConfig.OverrideNamespace == "" {
			registryValue = registryConfig.OverrideRegistry
		} else {
			registryValue = fmt.Sprintf("%s/%s", registryConfig.OverrideRegistry, registryConfig.OverrideNamespace)
		}
		args = append(args, fmt.Sprintf("registry=%s", registryValue))
	}

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
					Name:            "kotsadm-updater",
					Image:           fmt.Sprintf("kotsadm/kotsadm:%s", newVersion),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/scripts/kots-upgrade.sh"},
					Args:            args,
					Env: []corev1.EnvVar{
						{
							Name:  "DISABLE_OUTBOUND_CONNECTIONS",
							Value: os.Getenv("DISABLE_OUTBOUND_CONNECTIONS"),
						},
						{
							Name:  "KOTSADM_INSECURE_SRCREGISTRY",
							Value: os.Getenv("KOTSADM_INSECURE_SRCREGISTRY"),
						},
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

func findUpdatePod(ctx context.Context, clientset *kubernetes.Clientset) (*corev1.Pod, error) {
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

func getLastLogLineFromPod(ctx context.Context, clientset *kubernetes.Clientset, pod *corev1.Pod) (string, error) {
	if len(pod.Spec.Containers) == 0 {
		return "", nil
	}

	one := int64(1)
	podLogOpts := corev1.PodLogOptions{
		Follow:    false,
		Container: pod.Spec.Containers[0].Name,
		TailLines: &one,
	}

	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to get log stream")
	}
	defer podLogs.Close()

	var buffer bytes.Buffer
	byteWriter := bufio.NewWriter(&buffer)

	_, err = io.Copy(byteWriter, podLogs)
	if err != nil {
		return "", errors.Wrap(err, "failed to copy log")
	}
	return buffer.String(), nil
}
