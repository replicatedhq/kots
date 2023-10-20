package k8sutil

import (
	"context"
	"os"
	"time"

	"github.com/pkg/errors"
	types "github.com/replicatedhq/kots/pkg/k8sutil/types"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/segmentio/ksuid"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	KotsadmIDConfigMapName = "kotsadm-id"
)

func FindKotsadmImage(namespace string) (string, error) {
	clientset, err := GetClientset()
	if err != nil {
		return "", errors.Wrap(err, "failed to get k8s client set")
	}

	var containers []corev1.Container
	if os.Getenv("POD_OWNER_KIND") == "deployment" {
		kotsadmDeployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
		if err != nil {
			return "", errors.Wrap(err, "failed to get kotsadm deployment")
		}
		containers = kotsadmDeployment.Spec.Template.Spec.Containers
	} else {
		kotsadmStatefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
		if err != nil {
			return "", errors.Wrap(err, "failed to get kotsadm statefulset")
		}
		containers = kotsadmStatefulSet.Spec.Template.Spec.Containers
	}

	apiContainerIndex := -1
	for i, container := range containers {
		if container.Name == "kotsadm" {
			apiContainerIndex = i
			break
		}
	}

	if apiContainerIndex == -1 {
		return "", errors.New("kotsadm container not found")
	}

	kotsadmImage := containers[apiContainerIndex].Image

	return kotsadmImage, nil
}

// IsKotsadmClusterScoped will check if kotsadm has cluster scope access or not
func IsKotsadmClusterScoped(ctx context.Context, clientset kubernetes.Interface, namespace string) bool {
	rb, err := clientset.RbacV1().ClusterRoleBindings().Get(ctx, "kotsadm-rolebinding", metav1.GetOptions{})
	if err != nil {
		return false
	}
	for _, s := range rb.Subjects {
		if s.Kind != "ServiceAccount" {
			continue
		}
		if s.Name != "kotsadm" {
			continue
		}
		if s.Namespace != "" && s.Namespace == namespace {
			return true
		}
		if s.Namespace == "" && namespace == metav1.NamespaceDefault {
			return true
		}
	}
	return false
}

func GetKotsadmID(clientset kubernetes.Interface) string {
	var clusterID string
	configMap, err := GetKotsadmIDConfigMap(clientset)
	// if configmap is not found, generate a new guid and create a new configmap, if configmap is found, use the existing guid, otherwise generate
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		clusterID = ksuid.New().String()
	} else if configMap != nil {
		clusterID = configMap.Data["id"]
	} else {
		// configmap is missing for some reason, recreate with new guid, this will appear as a new instance in the report
		clusterID = ksuid.New().String()
		CreateKotsadmIDConfigMap(clientset, clusterID)
	}
	return clusterID
}

func GetKotsadmIDConfigMap(clientset kubernetes.Interface) (*corev1.ConfigMap, error) {
	namespace := util.PodNamespace
	existingConfigmap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), KotsadmIDConfigMapName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "failed to get configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		return nil, nil
	}
	return existingConfigmap, nil
}

func CreateKotsadmIDConfigMap(clientset kubernetes.Interface, kotsadmID string) error {
	var err error = nil
	configmap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      KotsadmIDConfigMapName,
			Namespace: util.PodNamespace,
			Labels: map[string]string{
				kotsadmtypes.KotsadmKey: kotsadmtypes.KotsadmLabelValue,
				kotsadmtypes.ExcludeKey: kotsadmtypes.ExcludeValue,
			},
		},
		Data: map[string]string{"id": kotsadmID},
	}
	_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Create(context.TODO(), &configmap, metav1.CreateOptions{})
	return err
}

func IsKotsadmIDConfigMapPresent() (bool, error) {
	clientset, err := GetClientset()
	if err != nil {
		return false, errors.Wrap(err, "failed to get clientset")
	}
	namespace := util.PodNamespace
	_, err = clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), KotsadmIDConfigMapName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return false, errors.Wrap(err, "failed to get configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		return false, nil
	}
	return true, nil
}

func UpdateKotsadmIDConfigMap(clientset kubernetes.Interface, kotsadmID string) error {
	namespace := util.PodNamespace
	existingConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), KotsadmIDConfigMapName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		return nil
	}
	if existingConfigMap.Data == nil {
		existingConfigMap.Data = map[string]string{}
	}
	existingConfigMap.Data["id"] = kotsadmID

	_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Update(context.Background(), existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}
	return nil
}

func FindKotsadm(clientset *kubernetes.Clientset, namespace string) (string, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=kotsadm"})
	if err != nil {
		return "", errors.Wrap(err, "failed to list pods")
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			return pod.Name, nil
		}
	}

	return "", errors.New("unable to find kotsadm pod")
}

func WaitForKotsadm(clientset kubernetes.Interface, namespace string, timeoutWaitingForWeb time.Duration) (string, error) {
	start := time.Now()

	for {
		// todo, find service, not pod
		pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=kotsadm"})
		if err != nil {
			return "", errors.Wrap(err, "failed to list pods")
		}

		readyPods := []corev1.Pod{}
		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodRunning {
				if pod.Status.ContainerStatuses[0].Ready {
					readyPods = append(readyPods, pod)
				}
			}
		}

		// kotsadm pods from different owners (deployment, statefulset) may co-exist for a brief period of time
		// during the upgrade process from versions pre 1.47.0 to 1.47+. we can't just check that the owner is a statefulset
		// because full snapshots taken before 1.47.0 will have kotsadm as a deployment and the restore will hang waiting for a statefulset.
		if len(readyPods) > 0 && PodsHaveTheSameOwner(readyPods) {
			return readyPods[0].Name, nil
		}

		time.Sleep(time.Second)

		if time.Since(start) > timeoutWaitingForWeb {
			return "", &types.ErrorTimeout{Message: "timeout waiting for kotsadm pod"}
		}
	}
}

func RestartKotsadm(ctx context.Context, clientset *kubernetes.Clientset, namespace string, timeout time.Duration) error {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: "app=kotsadm"})
	if err != nil {
		return errors.Wrap(err, "failed to list pods for termination")
	}

	deletedPods := make(map[string]bool)
	for _, pod := range pods.Items {
		err := clientset.CoreV1().Pods(namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to delete admin console")
		}
		deletedPods[pod.Name] = true
	}

	// wait for pods to stop running, or waiting for new pods will trip up.
	start := time.Now()
	for {
		pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: "app=kotsadm"})
		if err != nil {
			return errors.Wrap(err, "failed to list pods")
		}

		keepWaiting := false
		for _, pod := range pods.Items {
			if !deletedPods[pod.Name] {
				continue
			}

			if pod.Status.Phase == corev1.PodRunning {
				keepWaiting = true
				break
			}
		}

		if !keepWaiting {
			return nil
		}

		time.Sleep(time.Second)

		if time.Since(start) > timeout {
			return &types.ErrorTimeout{Message: "timeout waiting for kotsadm pod to stop"}
		}
	}
}

func DeleteKotsadm(ctx context.Context, clientset *kubernetes.Clientset, namespace string, isKurl bool) error {
	selectorLabels := map[string]string{
		kotsadmtypes.KotsadmKey: kotsadmtypes.KotsadmLabelValue,
	}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
	}

	// services (does not have a DeleteCollection method)
	services, err := clientset.CoreV1().Services(namespace).List(ctx, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to list services")
	}
	for _, service := range services.Items {
		err := clientset.CoreV1().Services(namespace).Delete(ctx, service.ObjectMeta.Name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to delete service %s in namespace %s", service.ObjectMeta.Name, service.ObjectMeta.Namespace)
		}
	}
	if err := waitForDeleteServices(ctx, clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete services")
	}

	// deployments
	err = clientset.AppsV1().Deployments(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm deployments")
	}
	if err := waitForDeleteDeployments(ctx, clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete deployments")
	}

	// statefulsets
	err = clientset.AppsV1().StatefulSets(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm statefulsets")
	}
	if err := waitForDeleteStatefulSets(ctx, clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete statefulsets")
	}

	// pods
	err = clientset.CoreV1().Pods(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm pods")
	}
	if err := waitForDeletePods(ctx, clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete pods")
	}

	// PVCs
	err = clientset.CoreV1().PersistentVolumeClaims(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm PVCs")
	}
	if err := waitForDeletePVCs(ctx, clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete PVCs")
	}

	// secrets
	err = clientset.CoreV1().Secrets(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm secrets")
	}
	if err := waitForDeleteSecrets(ctx, clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete secrets")
	}

	// configmaps
	err = clientset.CoreV1().ConfigMaps(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm configmaps")
	}
	if err := waitForDeleteConfigmaps(ctx, clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete configmaps")
	}

	// cluster role bindings
	err = clientset.RbacV1().ClusterRoleBindings().DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm clusterrolebindings")
	}
	if err := waitForDeleteClusterRoleBindings(ctx, clientset, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete clusterrolebindings")
	}

	// role bindings
	err = clientset.RbacV1().RoleBindings(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm rolebindings")
	}
	if err := waitForDeleteRoleBindings(ctx, clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete rolebindings")
	}

	// cluster roles
	err = clientset.RbacV1().ClusterRoles().DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm clusterroles")
	}
	if err := waitForDeleteClusterRoles(ctx, clientset, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete clusterroles")
	}

	// roles
	err = clientset.RbacV1().Roles(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm roles")
	}
	if err := waitForDeleteRoles(ctx, clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete roles")
	}

	// service accounts
	err = clientset.CoreV1().ServiceAccounts(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm serviceaccounts")
	}
	if err := waitForDeleteServiceAccounts(ctx, clientset, namespace, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete serviceaccounts")
	}

	if !isKurl {
		return nil
	}

	// kURL registry
	registryNS := "kurl"

	err = clientset.AppsV1().Deployments(registryNS).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete registry deployments")
	}
	if err := waitForDeleteDeployments(ctx, clientset, registryNS, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete registry deployments")
	}

	err = clientset.CoreV1().Pods(registryNS).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete registry pods")
	}
	if err := waitForDeletePods(ctx, clientset, registryNS, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete registry pods")
	}

	err = clientset.CoreV1().Secrets(registryNS).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to delete kotsadm secrets")
	}
	if err := waitForDeleteSecrets(ctx, clientset, registryNS, listOptions); err != nil {
		return errors.Wrap(err, "failed to wait for delete secrets")
	}

	return nil
}

func waitForDeleteServices(ctx context.Context, clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		services, err := clientset.CoreV1().Services(namespace).List(ctx, listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list services")
		}
		if len(services.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeleteDeployments(ctx context.Context, clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list deployments")
		}
		if len(deployments.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeleteStatefulSets(ctx context.Context, clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list statefulsets")
		}
		if len(statefulsets.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeletePods(ctx context.Context, clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list pods")
		}
		if len(pods.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeletePVCs(ctx context.Context, clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list pvcs")
		}
		if len(pvcs.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeleteSecrets(ctx context.Context, clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		secrets, err := clientset.CoreV1().Secrets(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list secrets")
		}
		if len(secrets.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeleteConfigmaps(ctx context.Context, clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		configmaps, err := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list configmaps")
		}
		if len(configmaps.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeleteClusterRoleBindings(ctx context.Context, clientset *kubernetes.Clientset, listOptions metav1.ListOptions) error {
	for {
		crbs, err := clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list clusterrolebindings")
		}
		if len(crbs.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeleteRoleBindings(ctx context.Context, clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		rbs, err := clientset.RbacV1().RoleBindings(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list rolebindings")
		}
		if len(rbs.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeleteClusterRoles(ctx context.Context, clientset *kubernetes.Clientset, listOptions metav1.ListOptions) error {
	for {
		crs, err := clientset.RbacV1().ClusterRoles().List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list clusterroles")
		}
		if len(crs.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeleteRoles(ctx context.Context, clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		roles, err := clientset.RbacV1().Roles(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list roles")
		}
		if len(roles.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitForDeleteServiceAccounts(ctx context.Context, clientset *kubernetes.Clientset, namespace string, listOptions metav1.ListOptions) error {
	for {
		serviceAccounts, err := clientset.CoreV1().ServiceAccounts(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list serviceaccounts")
		}
		if len(serviceAccounts.Items) == 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func GetKotsadmDeploymentUID(clientset kubernetes.Interface, namespace string) (apimachinerytypes.UID, error) {
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get replicated deployment")
	}

	return deployment.ObjectMeta.UID, nil
}
