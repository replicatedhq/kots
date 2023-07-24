package kotsadm

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	kotsadmresources "github.com/replicatedhq/kots/pkg/kotsadm/resources"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getKotsadmYAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	if deployOptions.IsMinimalRBAC {
		getKotsadmNamespacedRBAC(s, deployOptions.Namespace, deployOptions.Namespace, docs)
		for _, ns := range deployOptions.AdditionalNamespaces {
			getKotsadmNamespacedRBAC(s, ns, deployOptions.Namespace, docs)
		}
	} else {
		getKotsadmClusterRBAC(s, deployOptions.Namespace, docs)
	}

	var serviceAccount bytes.Buffer
	if err := s.Encode(kotsadmobjects.KotsadmServiceAccount(deployOptions.Namespace), &serviceAccount); err != nil {
		return nil, errors.Wrap(err, "failed to marshal kotsadm service account")
	}
	docs["kotsadm-serviceaccount.yaml"] = serviceAccount.Bytes()

	if deployOptions.IncludeMinio {
		kotsadmDeployment, err := kotsadmobjects.KotsadmDeployment(deployOptions)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get kotsadm deployment definition")
		}
		var deployment bytes.Buffer
		if err := s.Encode(kotsadmDeployment, &deployment); err != nil {
			return nil, errors.Wrap(err, "failed to marshal kotsadm deployment")
		}
		docs["kotsadm-deployment.yaml"] = deployment.Bytes()
	} else {
		size, err := getSize(deployOptions, "kotsadm", resource.MustParse("4Gi"))
		if err != nil {
			return nil, errors.Wrap(err, "failed to get size")
		}
		kotsadmSts, err := kotsadmobjects.KotsadmStatefulSet(deployOptions, size)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get kotsadm statefulset definition")
		}
		var statefulset bytes.Buffer
		if err := s.Encode(kotsadmSts, &statefulset); err != nil {
			return nil, errors.Wrap(err, "failed to marshal kotsadm statefulset")
		}
		docs["kotsadm-statefulset.yaml"] = statefulset.Bytes()
	}

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

func getKotsadmClusterRBAC(s *json.Serializer, namespace string, docs map[string][]byte) error {
	var role bytes.Buffer

	if err := s.Encode(kotsadmobjects.KotsadmClusterRole(), &role); err != nil {
		return errors.Wrap(err, "failed to marshal kotsadm role")
	}
	docs["kotsadm-role.yaml"] = role.Bytes()

	var roleBinding bytes.Buffer
	if err := s.Encode(kotsadmobjects.KotsadmClusterRoleBinding(namespace), &roleBinding); err != nil {
		return errors.Wrap(err, "failed to marshal kotsadm role binding")
	}
	docs["kotsadm-rolebinding.yaml"] = roleBinding.Bytes()
	return nil
}

func getKotsadmNamespacedRBAC(s *json.Serializer, additionalNamespace string, kotsadmNamespace string, docs map[string][]byte) error {
	var role bytes.Buffer

	if err := s.Encode(kotsadmobjects.KotsadmRole(additionalNamespace), &role); err != nil {
		return errors.Wrap(err, "failed to marshal kotsadm role")
	}
	roleName := fmt.Sprintf("kotsadm-role-%s.yaml", additionalNamespace)
	docs[roleName] = role.Bytes()

	var roleBinding bytes.Buffer
	if err := s.Encode(kotsadmobjects.KotsadmRoleBinding(additionalNamespace, kotsadmNamespace), &roleBinding); err != nil {
		return errors.Wrap(err, "failed to marshal kotsadm role binding")
	}
	roleBindingName := fmt.Sprintf("kotsadm-rolebinding-%s.yaml", additionalNamespace)
	docs[roleBindingName] = roleBinding.Bytes()
	return nil
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

	if deployOptions.IncludeMinio {
		if err := ensureKotsadmDeployment(*deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure kotsadm deployment")
		}
	} else {
		size, err := getSize(*deployOptions, "kotsadm", resource.MustParse("4Gi"))
		if err != nil {
			return errors.Wrap(err, "failed to get size")
		}
		if err := ensureKotsadmStatefulSet(*deployOptions, clientset, size); err != nil {
			return errors.Wrap(err, "failed to ensure kotsadm statefulset")
		}
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
	isClusterScoped, err := isKotsadmClusterScoped(&deployOptions)
	if err != nil {
		return errors.Wrap(err, "failed to check if kotsadm is cluster scoped")
	}

	// if this is cluster scoped, it's easy... create everything as a cluster role and cluster role binding
	// with pretty open permissions

	if isClusterScoped {
		return ensureKotsadmClusterRBAC(deployOptions, clientset)
	}

	// we want to ensure that the principle of least privilege is applied.
	// so we will create our role and rolebinding
	// and then create a role and role binding PER namespace that the application
	// wants...  everthing will be linked to the same service account

	if err := kotsadmresources.EnsureKotsadmRole(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm role")
	}

	if err := kotsadmresources.EnsureKotsadmRoleBinding(deployOptions.Namespace, deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm role binding")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(deployOptions.ApplicationMetadata, nil, nil)
	if err != nil {
		return errors.Wrap(err, "failed to decode application metadata")
	}
	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Application" {
		return errors.New("application metadata contained unepxected gvk")
	}

	application := obj.(*kotsv1beta1.Application)
	for _, additionalNamespace := range application.Spec.AdditionalNamespaces {
		if err = kotsadmresources.EnsureKotsadmRole(additionalNamespace, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure kotsadm additional namespace role")
		}

		if err = kotsadmresources.EnsureKotsadmRoleBinding(additionalNamespace, deployOptions.Namespace, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure kotsadm additional namespace role binding")
		}
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
	desiredDeployment, err := kotsadmobjects.KotsadmDeployment(deployOptions)
	if err != nil {
		return errors.Wrap(err, "failed to get desired kotsadm deployment definition")
	}

	existingDeployment, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing deployment")
		}

		_, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Create(context.TODO(), desiredDeployment, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}
		return nil
	}

	if err = kotsadmobjects.UpdateKotsadmDeployment(existingDeployment, desiredDeployment); err != nil {
		return errors.Wrap(err, "failed to merge deployments")
	}

	_, err = clientset.AppsV1().Deployments(deployOptions.Namespace).Update(context.TODO(), existingDeployment, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update kotsadm deployment")
	}

	return nil
}

func ensureKotsadmStatefulSet(deployOptions types.DeployOptions, clientset *kubernetes.Clientset, size resource.Quantity) error {
	desiredStatefulSet, err := kotsadmobjects.KotsadmStatefulSet(deployOptions, size)
	if err != nil {
		return errors.Wrap(err, "failed to get desired kotsadm statefulset definition")
	}

	existingStatefulSet, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing statefulset")
		}

		_, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Create(context.TODO(), desiredStatefulSet, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create statefulset")
		}

		return nil
	}

	if err = kotsadmobjects.UpdateKotsadmStatefulSet(existingStatefulSet, desiredStatefulSet); err != nil {
		return errors.Wrap(err, "failed to merge statefulsets")
	}

	_, err = clientset.AppsV1().StatefulSets(deployOptions.Namespace).Update(context.TODO(), existingStatefulSet, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update kotsadm statefulset")
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
func isKotsadmClusterScoped(deployOptions *types.DeployOptions) (bool, error) {
	if len(deployOptions.ApplicationMetadata) == 0 {
		return true, nil
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(deployOptions.ApplicationMetadata, nil, nil)
	if err != nil {
		return false, errors.Wrap(err, "failed to decode application metadata")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Application" {
		return false, errors.New("application metadata contained unepxected gvk")
	}

	application := obj.(*kotsv1beta1.Application)

	if deployOptions.UseMinimalRBAC && application.Spec.SupportMinimalRBACPrivileges {
		return false, nil
	}

	// An application can request cluster scope privileges quite simply
	if !application.Spec.RequireMinimalRBACPrivileges {
		return true, nil
	}

	for _, additionalNamespace := range application.Spec.AdditionalNamespaces {
		if additionalNamespace == "*" {
			return true, nil
		}
	}

	return false, nil
}
