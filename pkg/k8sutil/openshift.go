package k8sutil

import (
	"context"
	"strconv"
	"strings"

	"github.com/pkg/errors"
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
			if strings.HasPrefix(resource.GroupVersion, "apps.openshift.io/") {
				return true
			}
		}
	}
	return false
}

// GetOpenShiftPodSecurityContext returns a PodSecurityContext object that has:
// User set to the minimum value in the "openshift.io/sa.scc.uid-range" annotation.
// Group set to the minimum value in the "openshift.io/sa.scc.supplemental-groups" annotation if exists, else falls back to the minimum value in the "openshift.io/sa.scc.uid-range" annotation.
func GetOpenShiftPodSecurityContext(kotsadmNamespace string, strictSecurityContext bool) (*corev1.PodSecurityContext, error) {
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

	uid, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert uid to integer")
	}

	fsGroup, err := strconv.ParseInt(fsGroupStr, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert fsgroup to integer")
	}

	return SecurePodContext(uid, fsGroup, strictSecurityContext), nil
}
