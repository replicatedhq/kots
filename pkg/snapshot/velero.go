package snapshot

import (
	"fmt"
	"strings"
	"time"

	"context"
	"regexp"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmresources "github.com/replicatedhq/kots/pkg/kotsadm/resources"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/util"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/serverstatus"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	dockerImageNameRegex = regexp.MustCompile(`(?:([^\/]+)\/)?(?:([^\/]+)\/)?([^@:\/]+)(?:[@:](.+))`)
)

const (
	VeleroNamespaceConfigMapName = "kotsadm-velero-namespace"
)

type VeleroStatus struct {
	Version       string
	Plugins       []string
	Status        string
	Namespace     string
	VeleroPod     string
	NodeAgentPods []string

	NodeAgentVersion string
	NodeAgentStatus  string
}

func (s *VeleroStatus) ContainsPlugin(plugin string) bool {
	for _, x := range s.Plugins {
		if strings.Contains(x, plugin) {
			return true
		}
	}
	return false
}

func CheckKotsadmVeleroAccess(ctx context.Context, kotsadmNamespace string) (requiresAccess bool, finalErr error) {
	if kotsadmNamespace == "" {
		finalErr = errors.New("kotsadmNamespace param is required")
		return
	}

	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		finalErr = errors.Wrap(err, "failed to get cluster config")
		return
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		finalErr = errors.Wrap(err, "failed to create clientset")
		return
	}

	if k8sutil.IsKotsadmClusterScoped(ctx, clientset, kotsadmNamespace) {
		return
	}

	veleroConfigMap, err := clientset.CoreV1().ConfigMaps(kotsadmNamespace).Get(ctx, VeleroNamespaceConfigMapName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			finalErr = errors.Wrap(err, "failed to lookup velero configmap")
			return
		}
		// since this is a minimal rbac installation, kotsadm requires this configmap to know which namespace velero is installed in.
		// so if it's not found, then the user probably hasn't yet run the command that gives kotsadm access to the namespace velero is installed in,
		// which will also (re)generate this configmap
		requiresAccess = true
		return
	}

	if veleroConfigMap.Data == nil {
		requiresAccess = true
		return
	}

	veleroNamespace := veleroConfigMap.Data["veleroNamespace"]
	if veleroNamespace == "" {
		requiresAccess = true
		return
	}

	veleroClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		finalErr = errors.Wrap(err, "failed to create velero client")
		return
	}

	var backupStorageLocations velerov1.BackupStorageLocationList
	err = veleroClient.List(ctx, &backupStorageLocations, kbclient.InNamespace(veleroNamespace))
	if kuberneteserrors.IsForbidden(err) {
		requiresAccess = true
		return
	}
	if err != nil {
		finalErr = errors.Wrap(err, "failed to list backup storage locations")
		return
	}

	verifiedVeleroNamespace := ""
	for _, backupStorageLocation := range backupStorageLocations.Items {
		if backupStorageLocation.Name == "default" {
			verifiedVeleroNamespace = backupStorageLocation.Namespace
			break
		}
	}

	if verifiedVeleroNamespace == "" {
		requiresAccess = true
		return
	}

	_, err = clientset.RbacV1().Roles(verifiedVeleroNamespace).Get(ctx, "kotsadm-role", metav1.GetOptions{})
	if err != nil {
		requiresAccess = true
		return
	}

	_, err = clientset.RbacV1().RoleBindings(verifiedVeleroNamespace).Get(ctx, "kotsadm-rolebinding", metav1.GetOptions{})
	if err != nil {
		requiresAccess = true
		return
	}

	requiresAccess = false
	return
}

func EnsureVeleroPermissions(ctx context.Context, clientset kubernetes.Interface, veleroNamespace string, kotsadmNamespace string) error {
	veleroClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to create velero client")
	}

	var backupStorageLocations velerov1.BackupStorageLocationList
	err = veleroClient.List(ctx, &backupStorageLocations, kbclient.InNamespace(veleroNamespace))
	if err != nil {
		return errors.Wrapf(err, "failed to list backupstoragelocations in '%s' namespace", veleroNamespace)
	}

	verifiedVeleroNamespace := ""
	for _, backupStorageLocation := range backupStorageLocations.Items {
		if backupStorageLocation.Name == "default" {
			verifiedVeleroNamespace = backupStorageLocation.Namespace
			break
		}
	}

	if verifiedVeleroNamespace == "" {
		return errors.New(fmt.Sprintf("could not detect velero in '%s' namespace", veleroNamespace))
	}

	if err := kotsadmresources.EnsureKotsadmRole(verifiedVeleroNamespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm role")
	}

	if err := kotsadmresources.EnsureKotsadmRoleBinding(verifiedVeleroNamespace, kotsadmNamespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm rolebinding")
	}

	return nil
}

func EnsureVeleroNamespaceConfigMap(ctx context.Context, clientset kubernetes.Interface, veleroNamespace string, kotsadmNamespace string) error {
	existingConfigMap, err := clientset.CoreV1().ConfigMaps(kotsadmNamespace).Get(ctx, VeleroNamespaceConfigMapName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to lookup velero configmap")
		}

		newConfigMap := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      VeleroNamespaceConfigMapName,
				Namespace: kotsadmNamespace,
				Labels:    kotsadmtypes.GetKotsadmLabels(),
			},
			Data: map[string]string{
				"veleroNamespace": veleroNamespace,
			},
		}

		_, err := clientset.CoreV1().ConfigMaps(kotsadmNamespace).Create(ctx, newConfigMap, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create velero configmap")
		}

		return nil
	}

	if existingConfigMap.Data == nil {
		existingConfigMap.Data = make(map[string]string)
	}
	existingConfigMap.Data["veleroNamespace"] = veleroNamespace

	_, err = clientset.CoreV1().ConfigMaps(kotsadmNamespace).Update(ctx, existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update velero configmap")
	}

	return nil
}

// TryGetVeleroNamespaceFromConfigMap in the case of minimal rbac installations, a configmap containing the velero namespace
// will be created once the user gives kotsadm access to velero using the cli
func TryGetVeleroNamespaceFromConfigMap(ctx context.Context, clientset kubernetes.Interface, kotsadmNamespace string) string {
	c, err := clientset.CoreV1().ConfigMaps(kotsadmNamespace).Get(ctx, VeleroNamespaceConfigMapName, metav1.GetOptions{})
	if err != nil {
		return ""
	}
	if c.Data == nil {
		return ""
	}
	return c.Data["veleroNamespace"]
}

// DetectVeleroNamespace will detect and validate the velero namespace
// kotsadmNamespace is only required in minimal rbac installations. if empty, cluster scope privileges will be needed to detect and validate velero
func DetectVeleroNamespace(ctx context.Context, clientset kubernetes.Interface, kotsadmNamespace string) (string, error) {
	veleroNamespace := ""
	if kotsadmNamespace != "" {
		veleroNamespace = TryGetVeleroNamespaceFromConfigMap(ctx, clientset, kotsadmNamespace)
	}

	deployments, err := clientset.AppsV1().Deployments(veleroNamespace).List(ctx, metav1.ListOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return "", nil
	}

	if err != nil {
		// can't detect velero
		return "", nil
	}

	for _, deployment := range deployments.Items {
		if deployment.Name == "velero" {
			return deployment.Namespace, nil
		}
	}

	return "", nil
}

func DetectVelero(ctx context.Context, kotsadmNamespace string) (*VeleroStatus, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}

	veleroNamespace, err := DetectVeleroNamespace(ctx, clientset, kotsadmNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to detect velero namespace")
	}

	if veleroNamespace == "" {
		return nil, nil
	}

	veleroPod, err := getVeleroPod(ctx, clientset, veleroNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list velero pods")
	}

	nodeAgentPods, err := getNodeAgentPods(ctx, clientset, veleroNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list node-agent pods")
	}

	veleroStatus := VeleroStatus{
		Plugins:       []string{},
		Namespace:     veleroNamespace,
		VeleroPod:     veleroPod,
		NodeAgentPods: nodeAgentPods,
	}

	possibleDeployments, err := listPossibleVeleroDeployments(ctx, clientset, veleroNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list possible velero deployments")
	}

	version, err := getVersion(ctx, veleroNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get velero version")
	}
	veleroStatus.Version = version

	for _, deployment := range possibleDeployments {
		for _, initContainer := range deployment.Spec.Template.Spec.InitContainers {
			// the default installation is to name these like "velero-plugin-for-aws"
			veleroStatus.Plugins = append(veleroStatus.Plugins, initContainer.Name)
		}

		matches := dockerImageNameRegex.FindStringSubmatch(deployment.Spec.Template.Spec.Containers[0].Image)
		if len(matches) == 5 {
			status := "NotReady"

			if deployment.Status.AvailableReplicas > 0 {
				status = "Ready"
			}

			veleroStatus.Status = status
			goto DeploymentFound
		}
	}
DeploymentFound:

	daemonsets, err := listPossibleNodeAgentDaemonsets(ctx, clientset, veleroNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list node-agent daemonsets")
	}
	for _, daemonset := range daemonsets {
		matches := dockerImageNameRegex.FindStringSubmatch(daemonset.Spec.Template.Spec.Containers[0].Image)
		if len(matches) == 5 {
			status := "NotReady"

			if daemonset.Status.NumberAvailable > 0 {
				if daemonset.Status.NumberUnavailable == 0 {
					status = "Ready"
				}
			}
			veleroStatus.NodeAgentVersion = matches[4]
			veleroStatus.NodeAgentStatus = status

			goto NodeAgentFound
		}
	}
NodeAgentFound:

	return &veleroStatus, nil
}

func getVersion(ctx context.Context, namespace string) (string, error) {
	kbClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to get velero kube client")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	serverStatusGetter := &serverstatus.DefaultServerStatusGetter{
		Namespace: namespace,
		Context:   ctx,
	}
	serverStatus, err := serverStatusGetter.GetServerStatus(kbClient)
	if err != nil {
		return "", errors.Wrap(err, "error getting server version")
	}
	return serverStatus.Status.ServerVersion, nil
}

func getVeleroPod(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (string, error) {
	veleroLabels := map[string]string{
		"component": "velero",
		"deploy":    "velero",
	}
	labelSelector := labels.SelectorFromSet(veleroLabels)

	if util.IsEmbeddedCluster() {
		labelSelector = labels.SelectorFromSet(map[string]string{
			"name": "velero",
		})
	}

	veleroPods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to list velero pods before restarting")
	}

	for _, pod := range veleroPods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			if pod.Status.ContainerStatuses[0].Ready {
				return pod.Name, nil
			}
		}
	}

	return "", nil
}

func getNodeAgentPods(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	componentReq, err := labels.NewRequirement("component", selection.Equals, []string{"velero"})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create component requirement")
	}

	nameReq, err := labels.NewRequirement("name", selection.In, []string{"node-agent", "restic"})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create name requirement")
	}

	labelSelector := labels.NewSelector()
	labelSelector = labelSelector.Add(*componentReq, *nameReq)

	if util.IsEmbeddedCluster() {
		labelSelector = labels.SelectorFromSet(map[string]string{
			"name": "node-agent",
		})
	}

	nodeAgentPods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list node-agent pods before restarting")
	}

	pods := make([]string, 0)
	for _, pod := range nodeAgentPods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			if pod.Status.ContainerStatuses[0].Ready {
				pods = append(pods, pod.Name)
			}
		}
	}

	return pods, nil
}

// listPossibleVeleroDeployments filters with a label selector based on how we've found velero deployed
// using the CLI or the Helm Chart.
func listPossibleVeleroDeployments(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]v1.Deployment, error) {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "component=velero",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list deployments")
	}

	helmDeployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=velero",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list helm deployments")
	}

	return append(deployments.Items, helmDeployments.Items...), nil
}

// listPossibleNodeAgentDaemonsets filters with a label selector based on how we've found node-agent deployed
// using the CLI or the Helm Chart.
func listPossibleNodeAgentDaemonsets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]v1.DaemonSet, error) {
	daemonsets, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "component=velero",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list daemonsets")
	}

	helmDaemonsets, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=velero",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list helm daemonsets")
	}

	return append(daemonsets.Items, helmDaemonsets.Items...), nil
}

// restartVelero will restart velero (and node-agent)
func restartVelero(ctx context.Context, kotsadmNamespace string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s clientset")
	}

	veleroNamespace, err := DetectVeleroNamespace(ctx, clientset, kotsadmNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to detect velero namespace")
	}

	veleroDeployments, err := listPossibleVeleroDeployments(ctx, clientset, veleroNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to list possible velero deployments")
	}

	for _, veleroDeployment := range veleroDeployments {
		pods, err := clientset.CoreV1().Pods(veleroNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(veleroDeployment.Labels).String(),
		})
		if err != nil {
			return errors.Wrap(err, "failed to list velero pods")
		}

		for _, pod := range pods.Items {
			if err := clientset.CoreV1().Pods(veleroNamespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}); err != nil {
				return errors.Wrapf(err, "failed to delete %s pod", pod.Name)
			}
		}
	}

	nodeAgentDaemonSets, err := listPossibleNodeAgentDaemonsets(ctx, clientset, veleroNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to list possible node-agent daemonsets")
	}

	for _, nodeAgentDaemonSet := range nodeAgentDaemonSets {
		pods, err := clientset.CoreV1().Pods(veleroNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(nodeAgentDaemonSet.Labels).String(),
		})
		if err != nil {
			return errors.Wrap(err, "failed to list node-agent pods")
		}

		for _, pod := range pods.Items {
			if err := clientset.CoreV1().Pods(veleroNamespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}); err != nil {
				return errors.Wrapf(err, "failed to delete %s pod", pod.Name)
			}
		}
	}

	return nil
}
