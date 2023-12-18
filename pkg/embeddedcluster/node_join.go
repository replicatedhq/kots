package embeddedcluster

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/replicatedhq/kots/pkg/embeddedcluster/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/util"
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

const k0sTokenTemplate = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: %s
    server: https://%s:6443
  name: k0s
contexts:
- context:
    cluster: k0s
    user: kubelet-bootstrap
  name: k0s
current-context: k0s
kind: Config
users:
- name: kubelet-bootstrap
  user:
    token: %s
`

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

	newToken, err := makeK0sToken(ctx, client, nodeRole)
	if err != nil {
		return "", fmt.Errorf("failed to generate k0s token: %w", err)
	}

	now := time.Now()
	joinToken.Token = newToken
	joinToken.Creation = &now

	return newToken, nil
}

func makeK0sToken(ctx context.Context, client kubernetes.Interface, nodeRole string) (string, error) {
	rawToken, err := k8sutil.GenerateK0sBootstrapToken(client, time.Hour, nodeRole)
	if err != nil {
		return "", fmt.Errorf("failed to generate bootstrap token: %w", err)
	}

	cert, err := k8sutil.GetClusterCaCert(ctx, client)
	if err != nil {
		return "", fmt.Errorf("failed to get cluster ca cert: %w", err)
	}
	cert = base64.StdEncoding.EncodeToString([]byte(cert))

	firstPrimary, err := firstPrimaryIpAddress(ctx, client)
	if err != nil {
		return "", fmt.Errorf("failed to get first primary ip address: %w", err)
	}

	fullToken := fmt.Sprintf(k0sTokenTemplate, cert, firstPrimary, rawToken)
	gzipToken, err := util.GzipData([]byte(fullToken))
	if err != nil {
		return "", fmt.Errorf("failed to gzip token: %w", err)
	}
	b64Token := base64.StdEncoding.EncodeToString(gzipToken)

	return b64Token, nil
}

func firstPrimaryIpAddress(ctx context.Context, client kubernetes.Interface) (string, error) {
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list nodes: %w", err)
	}

	for _, node := range nodes.Items {
		if cp, ok := node.Labels["node-role.kubernetes.io/control-plane"]; !ok || cp != "true" {
			continue
		}

		for _, address := range node.Status.Addresses {
			if address.Type == "InternalIP" {
				return address.Address, nil
			}
		}

	}

	return "", fmt.Errorf("failed to find controller node")
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

	labels, err := getRolesNodeLabels(ctx, roles)
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

func getRolesNodeLabels(ctx context.Context, roles []string) (string, error) {
	roleListLabels := getRoleListLabels(roles)

	labels, err := getRoleNodeLabels(ctx, roles)
	if err != nil {
		return "", fmt.Errorf("failed to get node labels for roles %v: %w", roles, err)
	}
	roleLabels := append(roleListLabels, labels...)

	return strings.Join(roleLabels, ","), nil
}

// getRoleListLabels returns the labels needed to identify the roles of this node in the future
// one label will be the number of roles, and then deterministic label names will be used to store the role names
func getRoleListLabels(roles []string) []string {
	toReturn := []string{}
	toReturn = append(toReturn, fmt.Sprintf("%s=total-%d", types.EMBEDDED_CLUSTER_ROLE_LABEL, len(roles)))

	for idx, role := range roles {
		toReturn = append(toReturn, fmt.Sprintf("%s-%d=%s", types.EMBEDDED_CLUSTER_ROLE_LABEL, idx, labelify(role)))
	}

	return toReturn
}
