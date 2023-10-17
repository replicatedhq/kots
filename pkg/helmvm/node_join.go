package helmvm

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var addPrimaryNodeMut = sync.Mutex{}
var addSecondaryNodeMut = sync.Mutex{}
var primaryNodeJoinCommand []string
var primaryNodeJoinCommandCreation *time.Time
var secondaryNodeJoinCommand []string
var secondaryNodeJoinCommandCreation *time.Time

// GenerateAddNodeCommand will generate the HelmVM node add command for a primary or secondary node
// join commands will last for 24 hours, and will be cached for 1 hour after first generation
func GenerateAddNodeCommand(ctx context.Context, client kubernetes.Interface, primary bool) ([]string, *time.Time, error) {
	if primary {
		addPrimaryNodeMut.Lock()
		defer addPrimaryNodeMut.Unlock()
	} else {
		addSecondaryNodeMut.Lock()
		defer addSecondaryNodeMut.Unlock()
	}

	if primary {
		if primaryNodeJoinCommandCreation != nil && time.Now().Before(primaryNodeJoinCommandCreation.Add(time.Hour)) {
			expiry := primaryNodeJoinCommandCreation.Add(time.Hour * 24)
			return primaryNodeJoinCommand, &expiry, nil
		}
	} else {
		if secondaryNodeJoinCommandCreation != nil && time.Now().Before(secondaryNodeJoinCommandCreation.Add(time.Hour)) {
			expiry := secondaryNodeJoinCommandCreation.Add(time.Hour * 24)
			return secondaryNodeJoinCommand, &expiry, nil
		}
	}

	newCmd, err := runAddNodeCommandPod(ctx, client, primary)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to run add node command pod: %w", err)
	}

	now := time.Now()
	if primary {
		primaryNodeJoinCommand = newCmd
		primaryNodeJoinCommandCreation = &now
	} else {
		secondaryNodeJoinCommand = newCmd
		secondaryNodeJoinCommandCreation = &now
	}

	expiry := now.Add(time.Hour * 24)
	return newCmd, &expiry, nil
}

// run a pod that will generate the add node command
func runAddNodeCommandPod(ctx context.Context, client kubernetes.Interface, primary bool) ([]string, error) {
	podName := "k0s-token-generator"
	nodeRole := ""
	if primary {
		podName += "-primary"
		nodeRole = "controller"
	} else {
		podName += "-secondary"
		nodeRole = "worker"
	}

	// cleanup the pod if it already exists
	_ = client.CoreV1().Pods("kube-system").Delete(ctx, podName, metav1.DeleteOptions{})

	hostPathFile := corev1.HostPathFile
	hostPathDir := corev1.HostPathDirectory
	_, err := client.CoreV1().Pods("kube-system").Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: "kube-system",
			Labels: map[string]string{
				"replicated.app/embedded-cluster": "true",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyOnFailure,
			HostNetwork:   true,
			Volumes: []corev1.Volume{
				{
					Name: "bin",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/usr/local/bin/k0s",
							Type: &hostPathFile,
						},
					},
				},
				{
					Name: "lib",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/var/lib/k0s",
							Type: &hostPathDir,
						},
					},
				},
				{
					Name: "etc",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/etc/k0s",
							Type: &hostPathDir,
						},
					},
				},
				{
					Name: "run",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/run/k0s",
							Type: &hostPathDir,
						},
					},
				},
			},
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "node.k0sproject.io/role",
										Operator: corev1.NodeSelectorOpIn,
										Values: []string{
											"control-plane",
										},
									},
								},
							},
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:    "k0s-token-generator",
					Image:   "ubuntu:latest", // TODO use the kotsadm image here as we'll know it exists
					Command: []string{"/mnt/k0s"},
					Args: []string{
						"token",
						"create",
						"--expiry",
						"12h",
						"--role",
						nodeRole,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "bin",
							MountPath: "/mnt/k0s",
						},
						{
							Name:      "lib",
							MountPath: "/var/lib/k0s",
						},
						{
							Name:      "etc",
							MountPath: "/etc/k0s",
						},
						{
							Name:      "run",
							MountPath: "/run/k0s",
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create pod: %w", err)
	}

	// wait for the pod to complete
	for {
		pod, err := client.CoreV1().Pods("kube-system").Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get pod: %w", err)
		}

		if pod.Status.Phase == corev1.PodSucceeded {
			break
		}

		if pod.Status.Phase == corev1.PodFailed {
			return nil, fmt.Errorf("pod failed")
		}

		time.Sleep(time.Second)
	}

	// get the logs from the completed pod
	podLogs, err := client.CoreV1().Pods("kube-system").GetLogs(podName, &corev1.PodLogOptions{}).DoRaw(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod logs: %w", err)
	}

	// the logs are just a join token, which needs to be added to other things to get a join command
	token := string(podLogs)

	// for now, just return the token

	return []string{token}, nil
}
