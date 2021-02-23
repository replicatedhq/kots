package kotsadm

import (
	"bytes"
	"context"
	"time"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getKotsadmYAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var role bytes.Buffer
	if err := s.Encode(kotsadmobjects.KotsadmClusterRole(), &role); err != nil {
		return nil, errors.Wrap(err, "failed to marshal kotsadm role")
	}
	docs["kotsadm-role.yaml"] = role.Bytes()

	var roleBinding bytes.Buffer
	if err := s.Encode(kotsadmobjects.KotsadmClusterRoleBinding(deployOptions.Namespace), &roleBinding); err != nil {
		return nil, errors.Wrap(err, "failed to marshal kotsadm role binding")
	}
	docs["kotsadm-rolebinding.yaml"] = roleBinding.Bytes()

	var serviceAccount bytes.Buffer
	if err := s.Encode(kotsadmobjects.KotsadmServiceAccount(deployOptions.Namespace), &serviceAccount); err != nil {
		return nil, errors.Wrap(err, "failed to marshal kotsadm service account")
	}
	docs["kotsadm-serviceaccount.yaml"] = serviceAccount.Bytes()

	var deployment bytes.Buffer
	if err := s.Encode(kotsadmobjects.KotsadmDeployment(deployOptions), &deployment); err != nil {
		return nil, errors.Wrap(err, "failed to marshal kotsadm deployment")
	}
	docs["kotsadm-deployment.yaml"] = deployment.Bytes()

	var nodePort int32
	if deployOptions.IngressConfig.Spec.Enabled && deployOptions.IngressConfig.Spec.NodePort != nil {
		nodePort = int32(deployOptions.IngressConfig.Spec.NodePort.Port)
	}

	var service bytes.Buffer
	if err := s.Encode(kotsadmobjects.KotsadmService(deployOptions.Namespace, nodePort), &service); err != nil {
		return nil, errors.Wrap(err, "failed to marshal kotsadm service")
	}
	docs["kotsadm-service.yaml"] = service.Bytes()

	// TODO (ethan): ingress
	// TODO (ethan): identity-service

	return docs, nil
}

func waitForKotsadm(deployOptions *types.DeployOptions, previousDeployment *appsv1.Deployment, clientset *kubernetes.Clientset) error {
	start := time.Now()

	prevVersion := ""
	if previousDeployment != nil {
		prevVersion = previousDeployment.ObjectMeta.ResourceVersion
	}

	for {
		newDeployment, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to get new deployment")
		}

		if newDeployment.ObjectMeta.ResourceVersion != prevVersion {
			if newDeployment.Status.AvailableReplicas == newDeployment.Status.Replicas && newDeployment.Status.UnavailableReplicas == 0 {
				return nil
			}
		}

		time.Sleep(time.Second)

		if time.Now().Sub(start) > deployOptions.Timeout {
			return &types.ErrorTimeout{Message: "timeout waiting for kotsadm pod"}
		}
	}
}

func restartKotsadm(deployOptions *types.DeployOptions, clientset *kubernetes.Clientset) error {
	pods, err := clientset.CoreV1().Pods(deployOptions.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=kotsadm"})
	if err != nil {
		return errors.Wrap(err, "failed to list pods for termination")
	}

	deletedPods := make(map[string]bool)
	for _, pod := range pods.Items {
		err := clientset.CoreV1().Pods(deployOptions.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to delete admin console")
		}
		deletedPods[pod.Name] = true
	}

	// wait for pods to stop running, or waiting for new pods will trip up.
	start := time.Now()
	for {
		pods, err := clientset.CoreV1().Pods(deployOptions.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=kotsadm"})
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

		if time.Now().Sub(start) > deployOptions.Timeout {
			return &types.ErrorTimeout{Message: "timeout waiting for kotsadm pod to stop"}
		}
	}
}

func ensureKotsadmComponent(deployOptions *types.DeployOptions, clientset *kubernetes.Clientset) error {
	if deployOptions.EnsureRBAC {
		if err := ensureKotsadmRBAC(*deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure kotsadm rbac")
		}
	}

	if err := ensureApplicationMetadata(*deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure custom branding")
	}
	if err := ensureKotsadmDeployment(*deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm deployment")
	}

	var nodePort int32
	if deployOptions.IngressConfig.Spec.Enabled && deployOptions.IngressConfig.Spec.NodePort != nil {
		nodePort = int32(deployOptions.IngressConfig.Spec.NodePort.Port)
	}

	if err := ensureKotsadmService(deployOptions.Namespace, clientset, nodePort); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm service")
	}

	return nil
}

func ensureKotsadmRBAC(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	isClusterScoped, err := isKotsadmClusterScoped(deployOptions.ApplicationMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to check if kotsadm is cluster scoped")
	}

	if isClusterScoped {
		return ensureKotsadmClusterRBAC(deployOptions, clientset)
	}

	if err := EnsureKotsadmRole(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm role")
	}

	if err := EnsureKotsadmRoleBinding(deployOptions.Namespace, deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm role binding")
	}

	if err := ensureKotsadmServiceAccount(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm service account")
	}

	return nil
}

// ensureKotsadmClusterRBAC will ensure that the cluster role and cluster role bindings exists
func ensureKotsadmClusterRBAC(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	err := ensureKotsadmClusterRole(clientset)
	if err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm cluster role")
	}

	if err := ensureKotsadmClusterRoleBinding(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm cluster role binding")
	}

	if err := ensureKotsadmServiceAccount(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm service account")
	}

	return nil
}

func ensureKotsadmClusterRole(clientset *kubernetes.Clientset) error {
	_, err := clientset.RbacV1().ClusterRoles().Create(context.TODO(), kotsadmobjects.KotsadmClusterRole(), metav1.CreateOptions{})
	if err == nil || kuberneteserrors.IsAlreadyExists(err) {
		return nil
	}

	return errors.Wrap(err, "failed to create cluster role")
}

func ensureKotsadmClusterRoleBinding(serviceAccountNamespace string, clientset *kubernetes.Clientset) error {
	clusterRoleBinding, err := clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), "kotsadm-rolebinding", metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		_, err := clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), kotsadmobjects.KotsadmClusterRoleBinding(serviceAccountNamespace), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create cluster rolebinding")
		}
		return nil
	} else if err != nil {
		return errors.Wrap(err, "failed to get cluster rolebinding")
	}

	for _, subject := range clusterRoleBinding.Subjects {
		if subject.Namespace == serviceAccountNamespace && subject.Name == "kotsadm" && subject.Kind == "ServiceAccount" {
			return nil
		}
	}

	clusterRoleBinding.Subjects = append(clusterRoleBinding.Subjects, rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      "kotsadm",
		Namespace: serviceAccountNamespace,
	})

	_, err = clientset.RbacV1().ClusterRoleBindings().Update(context.TODO(), clusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create cluster rolebinding")
	}

	return nil
}

func EnsureKotsadmRole(namespace string, clientset *kubernetes.Clientset) error {
	role := kotsadmobjects.KotsadmRole(namespace)

	currentRole, err := clientset.RbacV1().Roles(namespace).Get(context.TODO(), "kotsadm-role", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get role")
		}

		_, err := clientset.RbacV1().Roles(namespace).Create(context.TODO(), role, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create role")
		}
		return nil
	}

	currentRole = updateKotsadmRole(currentRole, role)

	// we have now changed the role, so an upgrade is required
	_, err = clientset.RbacV1().Roles(namespace).Update(context.TODO(), currentRole, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update role")
	}

	return nil
}

func updateKotsadmRole(existing, desiredRole *rbacv1.Role) *rbacv1.Role {
	existing.Rules = desiredRole.Rules

	return existing
}

func EnsureKotsadmRoleBinding(roleBindingNamespace string, kotsadmNamespace string, clientset *kubernetes.Clientset) error {
	roleBinding := kotsadmobjects.KotsadmRoleBinding(roleBindingNamespace, kotsadmNamespace)

	currentRoleBinding, err := clientset.RbacV1().RoleBindings(roleBindingNamespace).Get(context.TODO(), "kotsadm-rolebinding", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get rolebinding")
		}

		_, err := clientset.RbacV1().RoleBindings(roleBindingNamespace).Create(context.TODO(), roleBinding, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create rolebinding")
		}
		return nil
	}

	currentRoleBinding = updateKotsadmRoleBinding(currentRoleBinding, roleBinding)

	// we have now changed the rolebinding, so an upgrade is required
	_, err = clientset.RbacV1().RoleBindings(roleBindingNamespace).Update(context.TODO(), currentRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update rolebinding")
	}

	return nil
}

func updateKotsadmRoleBinding(existing, desiredRoleBinding *rbacv1.RoleBinding) *rbacv1.RoleBinding {
	existing.Subjects = desiredRoleBinding.Subjects

	return existing
}

func ensureKotsadmServiceAccount(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get serviceaccouont")
		}

		_, err := clientset.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), kotsadmobjects.KotsadmServiceAccount(namespace), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create serviceaccount")
		}
	}

	return nil
}

func ensureKotsadmDeployment(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	existingDeployment, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing deployment")
		}

		_, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Create(context.TODO(), kotsadmobjects.KotsadmDeployment(deployOptions), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}
		return nil
	}

	if err = kotsadmobjects.UpdateKotsadmDeployment(existingDeployment, deployOptions); err != nil {
		return errors.Wrap(err, "failed to merge deployments")
	}

	_, err = clientset.AppsV1().Deployments(deployOptions.Namespace).Update(context.TODO(), existingDeployment, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update kotsadm deployment")
	}

	return nil
}

func ensureKotsadmService(namespace string, clientset *kubernetes.Clientset, nodePort int32) error {
	service := kotsadmobjects.KotsadmService(namespace, nodePort)

	existing, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		_, err := clientset.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create service")
		}

		return nil
	}

	existing = updateKotsadmService(existing, service)

	_, err = clientset.CoreV1().Services(namespace).Update(context.TODO(), existing, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update service")
	}

	return nil
}

func updateKotsadmService(existing, desiredService *corev1.Service) *corev1.Service {
	existing.Spec.Ports = desiredService.Spec.Ports

	return existing
}

// isKotsadmClusterScoped determines if the kotsadm pod should be running
// with cluster-wide permissions or not
func isKotsadmClusterScoped(applicationMetadata []byte) (bool, error) {
	if len(applicationMetadata) == 0 {
		return true, nil
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(applicationMetadata, nil, nil)
	if err != nil {
		return false, errors.Wrap(err, "failed to decode application metadata")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Application" {
		return false, errors.New("application metadata contained unepxected gvk")
	}

	application := obj.(*kotsv1beta1.Application)

	// An application can request cluster scope privileges quite simply
	if !application.Spec.RequireMinimalRBACPrivileges {
		return true, nil
	}

	return false, nil
}
