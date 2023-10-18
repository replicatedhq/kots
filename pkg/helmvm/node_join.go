package helmvm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type joinTokenEntry struct {
	Token    string
	Creation *time.Time
	Mut      sync.Mutex
}

var joinTokenMapMut = sync.Mutex{}
var joinTokenMap = map[string]*joinTokenEntry{}

// GenerateAddNodeToken will generate the HelmVM node add command for a primary or secondary node
// join commands will last for 24 hours, and will be cached for 1 hour after first generation
func GenerateAddNodeToken(ctx context.Context, client kubernetes.Interface, nodeRole string) (string, error) {
	// get the joinToken struct entry for this node role
	joinTokenMapMut.Lock()
	if _, ok := joinTokenMap[nodeRole]; !ok {
		joinTokenMap[nodeRole] = &joinTokenEntry{}
	}
	joinToken := joinTokenMap[nodeRole]
	joinTokenMapMut.Unlock()

	// lock the joinToken struct entry
	joinToken.Mut.Lock()
	defer joinToken.Mut.Unlock()

	// if the joinToken has been generated in the past hour, return it
	if joinToken.Creation != nil && time.Now().Before(joinToken.Creation.Add(time.Hour)) {
		return joinToken.Token, nil
	}

	newToken, err := runAddNodeCommandPod(ctx, client, nodeRole)
	if err != nil {
		return "", fmt.Errorf("failed to run add node command pod: %w", err)
	}

	now := time.Now()
	joinToken.Token = newToken
	joinToken.Creation = &now

	return newToken, nil
}

// run a pod that will generate the add node token
func runAddNodeCommandPod(ctx context.Context, client kubernetes.Interface, nodeRole string) (string, error) {
	podName := "k0s-token-generator-"
	suffix := strings.Replace(nodeRole, "+", "-", -1)
	podName += suffix

	// cleanup the pod if it already exists
	err := client.CoreV1().Pods("kube-system").Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return "", fmt.Errorf("failed to delete pod: %w", err)
		}
	}

	hostPathFile := corev1.HostPathFile
	hostPathDir := corev1.HostPathDirectory
	_, err = client.CoreV1().Pods("kube-system").Create(ctx, &corev1.Pod{
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
		return "", fmt.Errorf("failed to create pod: %w", err)
	}

	// wait for the pod to complete
	for {
		pod, err := client.CoreV1().Pods("kube-system").Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to get pod: %w", err)
		}

		if pod.Status.Phase == corev1.PodSucceeded {
			break
		}

		if pod.Status.Phase == corev1.PodFailed {
			return "", fmt.Errorf("pod failed")
		}

		time.Sleep(time.Second)
	}

	// get the logs from the completed pod
	podLogs, err := client.CoreV1().Pods("kube-system").GetLogs(podName, &corev1.PodLogOptions{}).DoRaw(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get pod logs: %w", err)
	}

	// the logs are just a join token, which needs to be added to other things to get a join command
	return string(podLogs), nil
}

// generate the add node command from the join token, the node roles, and info from the embedded-cluster-config configmap
func generateAddNodeCommand(ctx context.Context, client kubernetes.Interface, nodeRole string, token string) ([]string, error) {
	cm, err := ReadConfigMap(client)
	if err != nil {
		return nil, fmt.Errorf("failed to read configmap: %w", err)
	}

	clusterID := cm.Data["embedded-cluster-id"]
	binaryName := cm.Data["embedded-binary-name"]

	clusterUUID := uuid.UUID{}
	err = clusterUUID.UnmarshalText([]byte(clusterID))
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal cluster id %s: %w", clusterID, err)
	}

	fullToken := joinToken{
		ClusterID: clusterUUID,
		Token:     token,
		Role:      nodeRole,
	}

	b64token, err := fullToken.Encode()
	if err != nil {
		return nil, fmt.Errorf("unable to encode token: %w", err)
	}

	return []string{binaryName + " node join", b64token}, nil
}
