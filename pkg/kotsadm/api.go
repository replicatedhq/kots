package kotsadm

import (
	"bytes"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

var timeoutWaitingForAPI = time.Duration(time.Minute * 2)

func getApiYAML(namespace string) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var role bytes.Buffer
	if err := s.Encode(apiRole(namespace), &role); err != nil {
		return nil, errors.Wrap(err, "failed to marshal api role")
	}
	docs["api-role.yaml"] = role.Bytes()

	var roleBinding bytes.Buffer
	if err := s.Encode(apiRoleBinding(namespace), &roleBinding); err != nil {
		return nil, errors.Wrap(err, "failed to marshal api role binding")
	}
	docs["api-rolebinding.yaml"] = roleBinding.Bytes()

	var serviceAccount bytes.Buffer
	if err := s.Encode(apiServiceAccount(namespace), &serviceAccount); err != nil {
		return nil, errors.Wrap(err, "failed to marshal api service account")
	}
	docs["api-serviceaccount.yaml"] = serviceAccount.Bytes()

	var deployment bytes.Buffer
	if err := s.Encode(apiDeployment(namespace), &deployment); err != nil {
		return nil, errors.Wrap(err, "failed to marshal api deployment")
	}
	docs["api-deployment.yaml"] = deployment.Bytes()

	var service bytes.Buffer
	if err := s.Encode(apiService(namespace), &service); err != nil {
		return nil, errors.Wrap(err, "failed to marshal api service")
	}
	docs["api-service.yaml"] = service.Bytes()

	return docs, nil
}

func waitForAPI(deployOptions *DeployOptions, clientset *kubernetes.Clientset) error {
	start := time.Now()

	for {
		pods, err := clientset.CoreV1().Pods(deployOptions.Namespace).List(metav1.ListOptions{LabelSelector: "app=kotsadm-api"})
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

		if time.Now().Sub(start) > timeoutWaitingForAPI {
			return errors.New("timeout waiting for api pod")
		}
	}
}

func ensureAPI(deployOptions *DeployOptions, clientset *kubernetes.Clientset) error {
	if err := ensureApiRBAC(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api rbac")
	}

	if err := ensureApplicationMetadata(*deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure custom branding")
	}
	if err := ensureAPIDeployment(*deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api deployment")
	}

	if err := ensureAPIService(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api service")
	}

	return nil
}

func ensureApiRBAC(namespace string, clientset *kubernetes.Clientset) error {
	if err := ensureApiRole(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api role")
	}

	if err := ensureApiRoleBinding(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api role binding")
	}

	if err := ensureApiServiceAccount(namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api service account")
	}

	return nil
}

func ensureApiRole(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.RbacV1().Roles(namespace).Get("kotsadm-api-role", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get role")
		}

		_, err := clientset.RbacV1().Roles(namespace).Create(apiRole(namespace))
		if err != nil {
			return errors.Wrap(err, "failed to create role")
		}
	}

	return nil
}

func ensureApiRoleBinding(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.RbacV1().RoleBindings(namespace).Get("kotsadm-api-rolebinding", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get rolebinding")
		}

		_, err := clientset.RbacV1().RoleBindings(namespace).Create(apiRoleBinding(namespace))
		if err != nil {
			return errors.Wrap(err, "failed to create rolebinding")
		}
	}

	return nil
}

func ensureApiServiceAccount(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().ServiceAccounts(namespace).Get("kotsadm-api", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get serviceaccouont")
		}

		_, err := clientset.CoreV1().ServiceAccounts(namespace).Create(apiServiceAccount(namespace))
		if err != nil {
			return errors.Wrap(err, "failed to create serviceaccount")
		}
	}

	return nil
}

func ensureAPIDeployment(deployOptions DeployOptions, clientset *kubernetes.Clientset) error {
	_, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Get("kotsadm-api", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing deployment")
		}

		_, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Create(apiDeployment(deployOptions.Namespace))
		if err != nil {
			return errors.Wrap(err, "failed to create deployment")
		}
	}

	return nil
}

func ensureAPIService(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Services(namespace).Get("kotsadm-api", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		_, err := clientset.CoreV1().Services(namespace).Create(apiService(namespace))
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}
	}

	return nil
}
