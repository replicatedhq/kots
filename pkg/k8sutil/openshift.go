package k8sutil

import (
	"context"
	"strconv"
	"strings"

	configv1client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// IsOpenShift returns true if the cluster is positively identified as being an openshift cluster
func IsOpenShift(clientset kubernetes.Interface) bool {
	// ignore errors, since resources might be returned anyways
	// ignore groups, since we only need the data contained in resources
	_, resources, _ := clientset.Discovery().ServerGroupsAndResources()
	if resources != nil {
		for _, resource := range resources {
			if strings.Contains(resource.GroupVersion, "openshift") {
				return true
			}
		}
	}
	return false
}

// GetOpenShiftPodSecurityContext returns a PodSecurityContext object that has:
// RunAsUser set to the minimum value in the "openshift.io/sa.scc.uid-range" annotation.
// FSGroup set to the minimum value in the "openshift.io/sa.scc.supplemental-groups" annotation if exists, else falls back to the minimum value in the "openshift.io/sa.scc.uid-range" annotation.
func GetOpenShiftPodSecurityContext(kotsadmNamespace string) (*corev1.PodSecurityContext, error) {
	clientset, err := GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	ns, err := clientset.CoreV1().Namespaces().Get(context.TODO(), kotsadmNamespace, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get namespace")
	}

	// uid-range annotation must exist, otherwise the SCC will fail to create
	// reference: check the 1st note at the bottom of the page https://docs.openshift.com/enterprise/3.1/architecture/additional_concepts/authorization.html#admission
	uidRange, ok := ns.ObjectMeta.Annotations["openshift.io/sa.scc.uid-range"]
	if !ok {
		return nil, errors.New("annotation 'openshift.io/sa.scc.uid-range' not found")
	}

	supplementalGroups, ok := ns.ObjectMeta.Annotations["openshift.io/sa.scc.supplemental-groups"]
	if !ok {
		// fsgroup and supplemental groups strategies fall back to the uid-range annotation if the supplemental-groups annotation does not exist
		// reference: check the 1st note at the bottom of the page https://docs.openshift.com/enterprise/3.1/architecture/additional_concepts/authorization.html#admission
		supplementalGroups = uidRange
	}

	// supplemental groups annotation can contain multiple ranges separated by commas. get the first one.
	// reference: check the 3rd note at the bottom of the page https://docs.openshift.com/enterprise/3.1/architecture/additional_concepts/authorization.html#admission
	supplementalGroups = strings.Split(supplementalGroups, ",")[0]

	uidStr := strings.Split(uidRange, "/")[0]
	fsGroupStr := strings.Split(supplementalGroups, "/")[0] // use the minimum value of the supplemental groups range as fsgroup. reference: https://www.openshift.com/blog/a-guide-to-openshift-and-uids

	uid, err := strconv.Atoi(uidStr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert uid to integer")
	}

	fsGroup, err := strconv.Atoi(fsGroupStr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert fsgroup to integer")
	}

	psc := &corev1.PodSecurityContext{
		RunAsUser: util.IntPointer(uid),
		FSGroup:   util.IntPointer(fsGroup),
	}

	return psc, nil
}

func OpenShiftVersion() (string, error) {
	// openshift-apiserver does not report version,
	// clusteroperator/openshift-apiserver does, and only version number

	cfg, err := GetClusterConfig()
	if err != nil {
		return "", errors.Wrap(err, "failed to get cluster config")
	}

	client, err := configv1client.NewForConfig(cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to build client from config")
	}

	clusterOperator, err := client.ClusterOperators().Get(context.TODO(), "openshift-apiserver", metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get openshift apiserver")
	}

	if clusterOperator == nil {
		return "", errors.New("no openshift apiserver found")
	}

	for _, ver := range clusterOperator.Status.Versions {
		if ver.Name == "operator" {
			return ver.Version, nil
		}
	}

	return "", errors.New("no openshift operator found")
}
