package kurl

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	_ "embed"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//go:embed scripts/join-cert-gen.sh
var joinCertGenScript string

func createCertAndKey(ctx context.Context, client kubernetes.Interface, namespace string) (string, error) {
	configMap := getJoinCertGenConfigMapSpec(namespace)
	_, err := client.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to create configmap")
	}

	pod, err := getJoinCertGenPodSpec(client, namespace)
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

			if err := client.CoreV1().ConfigMaps(pod.Namespace).Delete(context.Background(), configMap.Name, metav1.DeleteOptions{}); err != nil {
				logger.Errorf("Failed to delete configmap %s: %v\n", configMap.Name, err)
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

func getJoinCertGenConfigMapSpec(namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-kurl-join-cert-gen",
			Namespace: namespace,
		},
		Data: map[string]string{
			"join-cert-gen.sh": joinCertGenScript,
		},
	}
}

func getJoinCertGenPodSpec(clientset kubernetes.Interface, namespace string) (*corev1.Pod, error) {
	var labels map[string]string
	var imagePullSecrets []corev1.LocalObjectReference
	var containers []corev1.Container

	if os.Getenv("POD_OWNER_KIND") == "deployment" {
		existingDeployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get existing deployment")
		}
		labels = existingDeployment.Labels
		imagePullSecrets = existingDeployment.Spec.Template.Spec.ImagePullSecrets
		containers = existingDeployment.Spec.Template.Spec.Containers
	} else {
		existingStatefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get existing statefulset")
		}
		labels = existingStatefulSet.Labels
		imagePullSecrets = existingStatefulSet.Spec.Template.Spec.ImagePullSecrets
		containers = existingStatefulSet.Spec.Template.Spec.Containers
	}

	apiContainerIndex := -1
	for i, container := range containers {
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

	binVolumeType := corev1.HostPathFile
	name := fmt.Sprintf("kurl-join-cert-%d", time.Now().Unix())
	scriptsFileMode := int32(0755)
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			SecurityContext: &securityContext,
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "node-role.kubernetes.io/control-plane",
										Operator: corev1.NodeSelectorOpExists,
									},
									{
										Key:      "node-role.kubernetes.io/master",
										Operator: corev1.NodeSelectorOpExists,
									},
								},
							},
						},
					},
				},
			},
			Tolerations: []corev1.Toleration{
				{
					Key:      "node-role.kubernetes.io/master",
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				},
				{
					Key:      "node-role.kubernetes.io/control-plane",
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				},
			},
			RestartPolicy:      corev1.RestartPolicyNever,
			ImagePullSecrets:   imagePullSecrets,
			ServiceAccountName: "kotsadm",
			Volumes: []corev1.Volume{
				{
					Name: "kubeadm",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/usr/bin/kubeadm",
							Type: &binVolumeType,
						},
					},
				},
				{
					Name: "scripts",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "kotsadm-kurl-join-cert-gen",
							},
							DefaultMode: &scriptsFileMode,
							Items: []corev1.KeyToPath{
								{
									Key:  "join-cert-gen.sh",
									Path: "join-cert-gen.sh",
								},
							},
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Image:           containers[apiContainerIndex].Image,
					ImagePullPolicy: corev1.PullNever,
					Name:            "join-cert-gen",
					Command:         []string{"/scripts/join-cert-gen.sh"},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "kubeadm",
							MountPath: "/usr/bin/kubeadm",
						},
						{
							Name:      "scripts",
							MountPath: "/scripts",
						},
					},
				},
			},
		},
	}

	return pod, nil
}
