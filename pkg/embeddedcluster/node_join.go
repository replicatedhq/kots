package embeddedcluster

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/replicatedhq/kots/pkg/embeddedcluster/types"
	"github.com/replicatedhq/kots/pkg/util"
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

// GenerateAddNodeToken will generate the embedded cluster node add command for a node with the specified roles
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
					Image:   "ubuntu:jammy", // this will not work on airgap, but it needs to be debian based at the moment
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

	// delete the completed pod
	err = client.CoreV1().Pods("kube-system").Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to delete pod: %w", err)
	}

	// the logs are just a join token, which needs to be added to other things to get a join command
	return string(podLogs), nil
}

// GenerateAddNodeCommand returns the command a user should run to add a node with the provided token
// the command will be of the form 'embeddedcluster node join ip:port UUID'
func GenerateAddNodeCommand(ctx context.Context, client kubernetes.Interface, token string) (string, error) {
	cm, err := ReadConfigMap(client)
	if err != nil {
		return "", fmt.Errorf("failed to read configmap: %w", err)
	}

	binaryName := cm.Data["embedded-binary-name"]

	// get the IP of a controller node
	nodeIP, err := getControllerNodeIP(ctx, client)
	if err != nil {
		return "", fmt.Errorf("failed to get controller node IP: %w", err)
	}

	// get the port of the 'admin-console' service
	port, err := getAdminConsolePort(ctx, client)
	if err != nil {
		return "", fmt.Errorf("failed to get admin console port: %w", err)
	}

	return fmt.Sprintf("sudo ./%s node join %s:%d %s", binaryName, nodeIP, port, token), nil
}

// GenerateK0sJoinCommand returns the k0s node join command, without the token but with all other required flags
// (including node labels generated from the roles etc)
func GenerateK0sJoinCommand(ctx context.Context, client kubernetes.Interface, roles []string) (string, error) {
	controllerRoleName, err := ControllerRoleName(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get controller role name: %w", err)
	}

	k0sRole := "worker"
	for _, role := range roles {
		if role == controllerRoleName {
			k0sRole = "controller"
		}
	}

	cmd := []string{"/usr/local/bin/k0s", "install", k0sRole}
	if k0sRole == "controller" {
		cmd = append(cmd, "--enable-worker")
	}

	labels, err := getRolesNodeLabels(ctx, client, roles)
	if err != nil {
		return "", fmt.Errorf("failed to get role labels: %w", err)
	}
	cmd = append(cmd, "--labels", labels)

	return strings.Join(cmd, " "), nil
}

// gets the port of the 'admin-console' service
func getAdminConsolePort(ctx context.Context, client kubernetes.Interface) (int32, error) {
	svc, err := client.CoreV1().Services(util.PodNamespace).Get(ctx, "admin-console", metav1.GetOptions{})
	if err != nil {
		return -1, fmt.Errorf("failed to get admin-console service: %w", err)
	}

	for _, port := range svc.Spec.Ports {
		if port.Name == "http" {
			return port.NodePort, nil
		}
	}
	return -1, fmt.Errorf("did not find port 'http' in service 'admin-console'")
}

// getControllerNodeIP gets the IP of a healthy controller node
func getControllerNodeIP(ctx context.Context, client kubernetes.Interface) (string, error) {
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list nodes: %w", err)
	}

	for _, node := range nodes.Items {
		if cp, ok := node.Labels["node-role.kubernetes.io/control-plane"]; !ok || cp != "true" {
			continue
		}

		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				for _, address := range node.Status.Addresses {
					if address.Type == "InternalIP" {
						return address.Address, nil
					}
				}
			}
		}

	}

	return "", fmt.Errorf("failed to find healthy controller node")
}

func getRolesNodeLabels(ctx context.Context, client kubernetes.Interface, roles []string) (string, error) {
	roleLabels := getRoleListLabels(roles)

	for _, role := range roles {
		labels, err := getRoleNodeLabels(ctx, client, role)
		if err != nil {
			return "", fmt.Errorf("failed to get node labels for role %s: %w", role, err)
		}
		roleLabels = append(roleLabels, labels...)
	}

	return strings.Join(roleLabels, ","), nil
}

// TODO: look up role in cluster config, apply additional labels based on role
func getRoleNodeLabels(ctx context.Context, client kubernetes.Interface, role string) ([]string, error) {
	toReturn := []string{}

	return toReturn, nil
}

// getRoleListLabels returns the labels needed to identify the roles of this node in the future
// one label will be the number of roles, and then deterministic label names will be used to store the role names
func getRoleListLabels(roles []string) []string {
	toReturn := []string{}
	toReturn = append(toReturn, fmt.Sprintf("%s=total-%d", types.EMBEDDED_CLUSTER_ROLE_LABEL, len(roles)))

	for idx, role := range roles {
		toReturn = append(toReturn, fmt.Sprintf("%s-%d=%s", types.EMBEDDED_CLUSTER_ROLE_LABEL, idx, role))
	}

	return toReturn
}
