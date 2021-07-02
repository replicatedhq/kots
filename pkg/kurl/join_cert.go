package kurl

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func createCertAndKey(ctx context.Context, client kubernetes.Interface, namespace string) (string, error) {
	pod, err := getPodSpec(client, namespace)
	if err != nil {
		return "", errors.Wrap(err, "failed to create pod spec")
	}

	_, err = client.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to create pod")
	}

	defer func() {
		go func() {
			// use context.background for the after-completion cleanup, as the parent context might already be over
			if err := client.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{}); err != nil {
				logger.Errorf("Failed to delete pod %s: %v\n", pod.Name, err)
			}
		}()
	}()

	for {
		status, err := client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		if err != nil {
			return "", errors.Wrap(err, "failed to get pod")
		}
		if status.Status.Phase == corev1.PodRunning ||
			status.Status.Phase == corev1.PodFailed ||
			status.Status.Phase == corev1.PodSucceeded {
			break
		}

		time.Sleep(time.Second * 1)

		// TODO: Do we need this?  Shouldn't Get function fail if there's a ctx error?
		if err := ctx.Err(); err != nil {
			return "", errors.Wrap(err, "failed to wait for pod to terminate")
		}
	}

	podLogs, err := getCertGenLogs(ctx, client, pod)
	if err != nil {
		return "", errors.Wrap(err, "failed to get pod logs")
	}

	key, err := parseCertGenOutput(podLogs)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse pod logs")
	}

	return key, nil
}

func getCertGenLogs(ctx context.Context, client kubernetes.Interface, pod *corev1.Pod) ([]byte, error) {
	podLogOpts := corev1.PodLogOptions{
		Follow:    true,
		Container: pod.Spec.Containers[0].Name,
	}

	req := client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
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
			return nil, errors.Wrap(resErr, "failed to copy log")
		} else {
			return buf.Bytes(), nil
		}
	case <-ctx.Done():
		return nil, errors.Wrap(ctx.Err(), "context ended copying log")
	}
}

func parseCertGenOutput(logs []byte) (string, error) {
	// Output looks like this:
	//
	// I0806 21:27:41.711156      41 version.go:251] remote version is much newer: v1.18.6; falling back to: stable-1.17
	// W0806 21:27:41.826204      41 validation.go:28] Cannot validate kube-proxy config - no validator is available
	// W0806 21:27:41.826231      41 validation.go:28] Cannot validate kubelet config - no validator is available
	// [upload-certs] Storing the certificates in Secret "kubeadm-certs" in the "kube-system" Namespace
	// [upload-certs] Using certificate key:
	// 7cf895f013c8977a24d3603c1802b78af146124d4c3223696b888a53352f4026

	scanner := bufio.NewScanner(bytes.NewReader(logs))
	foundKeyText := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if foundKeyText {
			return line, nil
		}
		if strings.Contains(line, "Using certificate key") {
			foundKeyText = true
		}
	}

	return "", fmt.Errorf("key not found in %d bytes of output", len(logs))
}

func getPodSpec(clientset kubernetes.Interface, namespace string) (*corev1.Pod, error) {
	existingStatefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get existing statefulset")
	}

	apiContainerIndex := -1
	for i, container := range existingStatefulSet.Spec.Template.Spec.Containers {
		if container.Name == "kotsadm" {
			apiContainerIndex = i
			break
		}
	}
	if apiContainerIndex == -1 {
		return nil, errors.New("kotsadm container not found")
	}

	securityContext := corev1.PodSecurityContext{
		RunAsUser: util.IntPointer(0),
	}

	configVolumeType := corev1.HostPathDirectory
	binVolumeType := corev1.HostPathFile
	name := fmt.Sprintf("kurl-join-cert-%d", time.Now().Unix())
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: existingStatefulSet.Namespace,
			Labels:    existingStatefulSet.Labels,
		},
		Spec: corev1.PodSpec{
			SecurityContext: &securityContext,
			NodeSelector: map[string]string{
				"node-role.kubernetes.io/master": "",
			},
			Tolerations: []corev1.Toleration{
				{
					Key:      "node-role.kubernetes.io/master",
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				},
			},
			RestartPolicy:    corev1.RestartPolicyNever,
			ImagePullSecrets: existingStatefulSet.Spec.Template.Spec.ImagePullSecrets,
			Volumes: []corev1.Volume{
				{
					Name: "config",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/etc/kubernetes/",
							Type: &configVolumeType,
						},
					},
				},
				{
					Name: "kubeadm",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/usr/bin/kubeadm",
							Type: &binVolumeType,
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Image:           existingStatefulSet.Spec.Template.Spec.Containers[apiContainerIndex].Image,
					ImagePullPolicy: corev1.PullNever,
					Name:            "join-cert-gen",
					Command:         []string{"/usr/bin/kubeadm"},
					Args:            []string{"init", "phase", "upload-certs", "--upload-certs"},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "config",
							MountPath: "/etc/kubernetes/",
						},
						{
							Name:      "kubeadm",
							MountPath: "/usr/bin/kubeadm",
						},
					},
				},
			},
		},
	}

	return pod, nil
}
