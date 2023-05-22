package appstate

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/appstate/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var (
	WaitForResourceFns = map[string]func(clientset kubernetes.Interface, namespace, name string) error{
		DaemonSetResourceKind:             WaitForDaemonSetToBeReady,
		DeploymentResourceKind:            WaitForDeploymentToBeReady,
		IngressResourceKind:               WaitForIngressToBeReady,
		PersistentVolumeClaimResourceKind: WaitForPersistentVolumeClaimToBeReady,
		ServiceResourceKind:               WaitForServiceToBeReady,
		StatefulSetResourceKind:           WaitForStatefulSetToBeReady,
	}
)

func WaitForResourceToBeReady(namespace, name string, gvr schema.GroupVersionResource, gvk *schema.GroupVersionKind) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	kind := ""
	if gvk != nil {
		kind = strings.ToLower(gvk.Kind)
	}

	if fn, ok := WaitForResourceFns[kind]; ok {
		return fn(clientset, namespace, name)
	}

	dynamicClientset, err := k8sutil.GetDynamicClient()
	if err != nil {
		return errors.Wrap(err, "failed to get dynamic clientset")
	}

	return WaitForGenericResourceToBeReady(dynamicClientset, namespace, name, gvr)
}

func WaitForDaemonSetToBeReady(clientset kubernetes.Interface, namespace, name string) error {
	for {
		r, err := clientset.AppsV1().DaemonSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing daemonset")
		}

		if !kuberneteserrors.IsNotFound(err) {
			state := CalculateDaemonSetState(clientset, namespace, r)
			if state == types.StateReady {
				return nil
			}
			logger.Debugf("daemonset %s in namespace %s is not ready, current state is %s", name, namespace, state)
		} else {
			logger.Debugf("daemonset %s in namespace %s is not ready, daemonset not found", name, namespace)
		}

		time.Sleep(time.Second * 2)
	}
}

func WaitForDeploymentToBeReady(clientset kubernetes.Interface, namespace, name string) error {
	for {
		r, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing deployment")
		}

		if !kuberneteserrors.IsNotFound(err) {
			state := CalculateDeploymentState(r)
			if state == types.StateReady {
				return nil
			}
			logger.Debugf("deployment %s in namespace %s is not ready, current state is %s", name, namespace, state)
		} else {
			logger.Debugf("deployment %s in namespace %s is not ready, deployment not found", name, namespace)
		}

		time.Sleep(time.Second * 2)
	}
}

func WaitForIngressToBeReady(clientset kubernetes.Interface, namespace, name string) error {
	for {
		r, err := clientset.NetworkingV1().Ingresses(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing ingress")
		}

		if !kuberneteserrors.IsNotFound(err) {
			state := CalculateIngressState(clientset, r)
			if state == types.StateReady {
				return nil
			}
			logger.Debugf("ingress %s in namespace %s is not ready, current state is %s", name, namespace, state)
		} else {
			logger.Debugf("ingress %s in namespace %s is not ready, ingress not found", name, namespace)
		}

		time.Sleep(time.Second * 2)
	}
}

func WaitForPersistentVolumeClaimToBeReady(clientset kubernetes.Interface, namespace, name string) error {
	for {
		r, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing persistentvolumeclaim")
		}

		if !kuberneteserrors.IsNotFound(err) {
			state := CalculatePersistentVolumeClaimState(r)
			if state == types.StateReady {
				return nil
			}
			logger.Debugf("persistentvolumeclaim %s in namespace %s is not ready, current state is %s", name, namespace, state)
		} else {
			logger.Debugf("persistentvolumeclaim %s in namespace %s is not ready, persistentvolumeclaim not found", name, namespace)
		}

		time.Sleep(time.Second * 2)
	}
}

func WaitForServiceToBeReady(clientset kubernetes.Interface, namespace, name string) error {
	for {
		r, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		if !kuberneteserrors.IsNotFound(err) {
			state := CalculateServiceState(clientset, r)
			if state == types.StateReady {
				return nil
			}
			logger.Debugf("service %s in namespace %s is not ready, current state is %s", name, namespace, state)
		} else {
			logger.Debugf("service %s in namespace %s is not ready, service not found", name, namespace)
		}

		time.Sleep(time.Second * 2)
	}
}

func WaitForStatefulSetToBeReady(clientset kubernetes.Interface, namespace, name string) error {
	for {
		r, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing statefulset")
		}

		if !kuberneteserrors.IsNotFound(err) {
			state := CalculateStatefulSetState(clientset, namespace, r)
			if state == types.StateReady {
				return nil
			}
			logger.Debugf("statefulset %s in namespace %s is not ready, current state is %s", name, namespace, state)
		} else {
			logger.Debugf("statefulset %s in namespace %s is not ready, statefulset not found", name, namespace)
		}

		time.Sleep(time.Second * 2)
	}
}

func WaitForGenericResourceToBeReady(dynamicClientset dynamic.Interface, namespace, name string, gvr schema.GroupVersionResource) error {
	for {
		_, err := dynamicClientset.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing resource")
		}

		if kuberneteserrors.IsNotFound(err) {
			logger.Debugf("resource %s in namespace %s is not ready, resource not found", name, namespace)
		} else {
			logger.Debugf("resource %s in namespace %s is ready", name, namespace)
		}

		time.Sleep(time.Second * 2)
	}
}
