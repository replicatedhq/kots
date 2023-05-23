package appstate

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/appstate/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/jsonpath"
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

func WaitForResourceToBeReady(namespace, name string, gvk *schema.GroupVersionKind) error {
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

	dr, err := k8sutil.GetDynamicResourceInterface(gvk, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to get dynamic resource interface")
	}

	return WaitForGenericResourceToBeReady(dr, name)
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
		}

		time.Sleep(time.Second * 2)
	}
}

func WaitForGenericResourceToBeReady(dr dynamic.ResourceInterface, name string) (err error) {
	for {
		_, err := dr.Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing resource")
		}

		if !kuberneteserrors.IsNotFound(err) {
			return nil
		}

		time.Sleep(time.Second * 2)
	}
}

func WaitForProperty(namespace, name string, gvk *schema.GroupVersionKind, path, desiredValue string) error {
	dr, err := k8sutil.GetDynamicResourceInterface(gvk, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to get dynamic resource interface")
	}

	for {
		r, err := dr.Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing resource")
		}

		if !kuberneteserrors.IsNotFound(err) {
			if matches, err := resourcePropertyMatchesValue(r, path, desiredValue); err != nil {
				return errors.Wrap(err, "failed to check if resource property matches value")
			} else if matches {
				return nil
			}
		}

		time.Sleep(time.Second * 2)
	}
}

func resourcePropertyMatchesValue(r *unstructured.Unstructured, path, desiredValue string) (bool, error) {
	if path == "" {
		return false, errors.New("key cannot be empty")
	}

	parser := jsonpath.New("wait-for-property")
	if err := parser.Parse(fmt.Sprintf("{ %s }", path)); err != nil {
		return false, errors.Wrapf(err, "failed to parse jsonpath %s", path)
	}

	buf := new(bytes.Buffer)
	err := parser.Execute(buf, r.Object)
	if err != nil {
		// don't return an error here since the field may not exist yet
		logger.Warnf("failed to execute jsonpath: %v", err)
	}

	if buf.String() == desiredValue {
		return true, nil
	}

	return false, nil
}
