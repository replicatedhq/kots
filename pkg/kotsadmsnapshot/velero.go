package snapshot

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8s"
	"github.com/replicatedhq/kots/pkg/logger"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	v1 "k8s.io/api/apps/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	dockerImageNameRegex = regexp.MustCompile("(?:([^\\/]+)\\/)?(?:([^\\/]+)\\/)?([^@:\\/]+)(?:[@:](.+))")
)

type VeleroStatus struct {
	Version   string
	Plugins   []string
	Status    string
	Namespace string

	ResticVersion string
	ResticStatus  string
}

func CheckKotsadmVeleroAccess() (requiresAccess bool, finalErr error) {
	cfg, err := config.GetConfig()
	if err != nil {
		finalErr = errors.Wrap(err, "failed to get cluster config")
		return
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		finalErr = errors.Wrap(err, "failed to create clientset")
		return
	}

	if k8s.IsKotsadmClusterScoped(context.TODO(), clientset) {
		return
	}

	veleroConfigMap, err := clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), "kotsadm-velero-namespace", metav1.GetOptions{})
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

	veleroNamespace := veleroConfigMap.Data["veleroNamespace"]

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		finalErr = errors.Wrap(err, "failed to create velero clientset")
		return
	}

	backupStorageLocations, err := veleroClient.BackupStorageLocations(veleroNamespace).List(context.TODO(), metav1.ListOptions{})
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
		logger.Error(errors.New(fmt.Sprintf("could not detect velero in '%s' namespace", veleroNamespace)))
		requiresAccess = true
		return
	}

	_, err = clientset.RbacV1().Roles(verifiedVeleroNamespace).Get(context.TODO(), "kotsadm-role", metav1.GetOptions{})
	if err != nil {
		requiresAccess = true
		return
	}

	_, err = clientset.RbacV1().RoleBindings(verifiedVeleroNamespace).Get(context.TODO(), "kotsadm-rolebinding", metav1.GetOptions{})
	if err != nil {
		requiresAccess = true
		return
	}

	requiresAccess = false
	return
}

func getMinimalRBACVeleroNamespace(clientset kubernetes.Interface) (string, error) {
	// a configmap which includes the namespace in which Velero is installed should already be created in minimal rbac mode
	c, err := clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), "kotsadm-velero-namespace", metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return "", nil
		}
		return "", errors.Wrap(err, "failed to get velero configmap")
	}
	return c.Data["veleroNamespace"], nil
}

func DetectVeleroNamespace() (string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "", errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to create clientset")
	}

	veleroNamespace := ""
	if !k8s.IsKotsadmClusterScoped(context.TODO(), clientset) {
		veleroNamespace, err = getMinimalRBACVeleroNamespace(clientset)
		if err != nil {
			return "", nil
		}
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to create velero clientset")
	}

	backupStorageLocations, err := veleroClient.BackupStorageLocations(veleroNamespace).List(context.TODO(), metav1.ListOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return "", nil
	}

	if err != nil {
		// can't detect velero
		return "", nil
	}

	for _, backupStorageLocation := range backupStorageLocations.Items {
		if backupStorageLocation.Name == "default" {
			return backupStorageLocation.Namespace, nil
		}
	}

	return "", nil
}

func DetectVelero() (*VeleroStatus, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	veleroNamespace, err := DetectVeleroNamespace()
	if err != nil {
		return nil, errors.Wrap(err, "failed to detect velero namespace")
	}

	if veleroNamespace == "" {
		return nil, nil
	}

	veleroStatus := VeleroStatus{
		Plugins:   []string{},
		Namespace: veleroNamespace,
	}

	possibleDeployments, err := listPossibleVeleroDeployments(clientset, veleroNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list possible velero deployments")
	}

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

			veleroStatus.Version = matches[4]
			veleroStatus.Status = status

			goto DeploymentFound
		}
	}
DeploymentFound:

	daemonsets, err := listPossibleResticDaemonsets(clientset, veleroNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list restic daemonsets")
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

			veleroStatus.ResticVersion = matches[4]
			veleroStatus.ResticStatus = status

			goto ResticFound
		}
	}
ResticFound:

	return &veleroStatus, nil
}

// listPossibleVeleroDeployments filters with a label selector based on how we've found velero deployed
// using the CLI or the Helm Chart.
func listPossibleVeleroDeployments(clientset *kubernetes.Clientset, namespace string) ([]v1.Deployment, error) {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "component=velero",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list deployments")
	}

	helmDeployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=velero",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list helm deployments")
	}

	return append(deployments.Items, helmDeployments.Items...), nil
}

// listPossibleResticDaemonsets filters with a label selector based on how we've found restic deployed
// using the CLI or the Helm Chart.
func listPossibleResticDaemonsets(clientset *kubernetes.Clientset, namespace string) ([]v1.DaemonSet, error) {
	daemonsets, err := clientset.AppsV1().DaemonSets(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "component=velero",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list daemonsets")
	}

	helmDaemonsets, err := clientset.AppsV1().DaemonSets(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=velero",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list helm daemonsets")
	}

	return append(daemonsets.Items, helmDaemonsets.Items...), nil
}

// RestartVelero will restart velero (and restic)
func RestartVelero() error {
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create clientset")
	}

	namespace, err := DetectVeleroNamespace()
	if err != nil {
		return errors.Wrap(err, "failed to detect velero namespace")
	}

	veleroDeployments, err := listPossibleVeleroDeployments(clientset, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to list velero deployments")
	}

	for _, veleroDeployment := range veleroDeployments {
		pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(veleroDeployment.Labels).String(),
		})
		if err != nil {
			return errors.Wrap(err, "failed to list pods in velero deployment")
		}

		for _, pod := range pods.Items {
			if err := clientset.CoreV1().Pods(namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{}); err != nil {
				return errors.Wrap(err, "failed to delete velero deployment")
			}

		}
	}

	resticDaemonSets, err := listPossibleResticDaemonsets(clientset, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to list restic daemonsets")
	}

	for _, resticDaemonSet := range resticDaemonSets {
		pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(resticDaemonSet.Labels).String(),
		})
		if err != nil {
			return errors.Wrap(err, "failed to list pods in restic daemonset")
		}

		for _, pod := range pods.Items {
			if err := clientset.CoreV1().Pods(namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{}); err != nil {
				return errors.Wrap(err, "failed to delete restic daemonset")
			}

		}
	}

	return nil
}
