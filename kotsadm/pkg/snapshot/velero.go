package snapshot

import (
	"context"
	"regexp"

	"github.com/pkg/errors"
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
	Version string
	Plugins []string
	Status  string

	ResticVersion string
	ResticStatus  string
}

func DetectVeleroNamespace() (string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "", errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to create velero clientset")
	}

	backupStorageLocations, err := veleroClient.BackupStorageLocations("").List(context.TODO(), metav1.ListOptions{})
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
		Plugins: []string{},
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
