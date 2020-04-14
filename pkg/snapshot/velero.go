package snapshot

import (
	"regexp"

	"github.com/pkg/errors"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	dockerImageNameRegex = regexp.MustCompile("(?:([^\\/]+)\\/)?(?:([^\\/]+)\\/)?([^@:\\/]+)(?:[@:](.+))")
)

func DetectVelero() (string, []string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to create clientset")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to create velero clientset")
	}

	backupStorageLocations, err := veleroClient.BackupStorageLocations("").List(metav1.ListOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return "", nil, nil
	}

	if err != nil {
		// can't detect velero
		return "", nil, nil
	}

	veleroNamespace := ""
	for _, backupStorageLocation := range backupStorageLocations.Items {
		if backupStorageLocation.Name == "default" {
			veleroNamespace = backupStorageLocation.Namespace
		}
	}

	if veleroNamespace == "" {
		return "", nil, nil
	}

	deployments, err := clientset.AppsV1().Deployments(veleroNamespace).List(metav1.ListOptions{
		LabelSelector: "component=velero",
	})
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to list deployments")
	}

	plugins := []string{}
	for _, deployment := range deployments.Items {
		for _, initContainer := range deployment.Spec.Template.Spec.InitContainers {
			// the default installation is to name these like "velero-plugin-for-aws"
			plugins = append(plugins, initContainer.Name)
		}

		matches := dockerImageNameRegex.FindStringSubmatch(deployment.Spec.Template.Spec.Containers[0].Image)
		if len(matches) == 5 {
			return matches[4], plugins, nil
		}

	}

	// get here, no velero
	return "", nil, nil
}
