package kotsadm

import (
	"bytes"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

var timeoutWaitingForKotsadm = time.Duration(time.Minute * 2)

func getKotsadmYAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var role bytes.Buffer
	if err := s.Encode(kotsadmRole(deployOptions.Namespace), &role); err != nil {
		return nil, errors.Wrap(err, "failed to marshal kotsadm role")
	}
	docs["kotsadm-role.yaml"] = role.Bytes()

	var roleBinding bytes.Buffer
	if err := s.Encode(kotsadmRoleBinding(deployOptions.Namespace), &roleBinding); err != nil {
		return nil, errors.Wrap(err, "failed to marshal kotsadm role binding")
	}
	docs["kotsadm-rolebinding.yaml"] = roleBinding.Bytes()

	var serviceAccount bytes.Buffer
	if err := s.Encode(kotsadmServiceAccount(deployOptions.Namespace), &serviceAccount); err != nil {
		return nil, errors.Wrap(err, "failed to marshal kotsadm service account")
	}
	docs["kotsadm-serviceaccount.yaml"] = serviceAccount.Bytes()

	var deployment bytes.Buffer
	if err := s.Encode(kotsadmDeployment(deployOptions), &deployment); err != nil {
		return nil, errors.Wrap(err, "failed to marshal kotsadm deployment")
	}
	docs["kotsadm-deployment.yaml"] = deployment.Bytes()

	var service bytes.Buffer
	if err := s.Encode(kotsadmService(deployOptions.Namespace), &service); err != nil {
		return nil, errors.Wrap(err, "failed to marshal kotsadm service")
	}
	docs["kotsadm-service.yaml"] = service.Bytes()

	return docs, nil
}

func waitForKotsadm(deployOptions *types.DeployOptions, clientset *kubernetes.Clientset) error {
	start := time.Now()

	for {
		pods, err := clientset.CoreV1().Pods(deployOptions.Namespace).List(metav1.ListOptions{LabelSelector: "app=kotsadm"})
		if err != nil {
			return errors.Wrap(err, "failed to list pods")
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodRunning {
				if pod.Status.ContainerStatuses[0].Ready == true {
					return nil
				}
			}
		}

		time.Sleep(time.Second)

		if time.Now().Sub(start) > timeoutWaitingForKotsadm {
			return errors.New("timeout waiting for kotsadm pod")
		}
	}
}

func ensureKotsadmComponent(deployOptions *types.DeployOptions, clientset *kubernetes.Clientset) error {
	if err := ensureKotsadmRBAC(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm rbac")
	}

	if err := ensureApplicationMetadata(*deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure custom branding")
	}
	if err := ensureKotsadmDeployment(*deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm deployment")
	}

	if err := ensureKotsadmService(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm service")
	}

	return nil
}

func ensureKotsadmRBAC(namespace string, clientset *kubernetes.Clientset) error {
	if err := ensureKotsadmRole(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm role")
	}

	if err := ensureKotsadmRoleBinding(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm role binding")
	}

	if err := ensureKotsadmServiceAccount(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm service account")
	}

	return nil
}

func ensureKotsadmRole(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.RbacV1().Roles(namespace).Get("kotsadm-role", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get role")
		}

		_, err := clientset.RbacV1().Roles(namespace).Create(kotsadmRole(namespace))
		if err != nil {
			return errors.Wrap(err, "failed to create role")
		}
	}

	return nil
}

func ensureKotsadmRoleBinding(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.RbacV1().RoleBindings(namespace).Get("kotsadm-rolebinding", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get rolebinding")
		}

		_, err := clientset.RbacV1().RoleBindings(namespace).Create(kotsadmRoleBinding(namespace))
		if err != nil {
			return errors.Wrap(err, "failed to create rolebinding")
		}
	}

	return nil
}

func ensureKotsadmServiceAccount(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().ServiceAccounts(namespace).Get("kotsadm", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get serviceaccouont")
		}

		_, err := clientset.CoreV1().ServiceAccounts(namespace).Create(kotsadmServiceAccount(namespace))
		if err != nil {
			return errors.Wrap(err, "failed to create serviceaccount")
		}
	}

	return nil
}

func ensureKotsadmDeployment(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	_, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Get("kotsadm", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing deployment")
		}

		_, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Create(kotsadmDeployment(deployOptions))
		if err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}
	}

	return nil
}

func ensureKotsadmService(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Services(namespace).Get("kotsadm", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		_, err := clientset.CoreV1().Services(namespace).Create(kotsadmService(namespace))
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}
	}

	return nil
}
