package client

import (
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	core "k8s.io/client-go/testing"
)

const (
	podManifest = `apiVersion: v1
kind: Pod
metadata:
  name: test
  namespace: default
`
	rabbitmqCRManifest = `apiVersion: rabbitmq.com/v1beta1
kind: RabbitmqCluster
metadata:
  name: rabbitmq
  namespace: default
spec:
  rabbitmq:
    image: rabbitmq:3.8.2-management-alpine	
`
)

var (
	unstructuredPod = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      "test",
				"namespace": "default",
				"annotations": map[string]interface{}{
					"kots.io/app-slug": "test",
				},
				"labels": map[string]interface{}{
					"label/restore": "true",
				},
			},
		},
	}
	unstructuredRabbitMQCR = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rabbitmq.com/v1beta1",
			"kind":       "RabbitmqCluster",
			"metadata": map[string]interface{}{
				"name":      "rabbitmq",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"rabbitmq": map[string]interface{}{
					"image": "rabbitmq:3.8.2-management-alpine",
				},
			},
		},
	}
	unstructuredRabbitMQCRD = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apiextensions.k8s.io",
			"kind":       "CustomResourceDefinition",
			"metadata": map[string]interface{}{
				"name":      "rabbitmq-crd",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"group": "rabbitmq.com",
				"names": map[string]interface{}{
					"kind":     "RabbitmqCluster",
					"listKind": "RabbitmqClusterList",
					"plural":   "rabbitmqclusters",
					"singular": "rabbitmqcluster",
				},
				"versions": []interface{}{
					map[string]interface{}{
						"name": "v1beta1",
					},
				},
			},
		},
	}
	unstructuredPodMarkedDeletion = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":              "test-deleting",
				"namespace":         "default",
				"deletionTimestamp": "2020-04-20T15:20:00Z",
				"annotations": map[string]interface{}{
					"kots.io/app-slug": "test",
				},
			},
		},
	}
	unstructuredPodExcludeFromBackup = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      "test-deleting",
				"namespace": "default",
				"labels": map[string]interface{}{
					"velero.io/exclude-from-backup": "true",
				},
			},
		},
	}
	unstructuredPodWithRestoreLabel = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      "test-restore-label-match",
				"namespace": "default",
				"labels": map[string]interface{}{
					"label/restore": "true",
				},
				"annotations": map[string]interface{}{
					"kots.io/app-slug": "test",
				},
			},
		},
	}
	unstructuredPodWithRestoreLabelNotMatch = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      "test-deleting",
				"namespace": "default",
				"labels": map[string]interface{}{
					"label/restore": "false",
				},
			},
		},
	}
)

var (
	podGVK = unstructuredPod.GroupVersionKind()
	crdGVK = unstructuredRabbitMQCRD.GroupVersionKind()
	crGVK  = unstructuredRabbitMQCR.GroupVersionKind()
)

var (
	podGVR = unstructuredPod.GroupVersionKind().GroupVersion().WithResource("pods")
)

// Mocks for testing
var kubectlApplierMock = K8sApplierMock{}

type K8sApplierMock struct {
}

func (k *K8sApplierMock) Apply(targetNamespace string, slug string, yamlDoc []byte, dryRun bool, wait bool, annotateSlug bool) ([]byte, []byte, error) {
	return nil, nil, nil
}
func (k *K8sApplierMock) Remove(targetNamespace string, yamlDoc []byte, wait bool) ([]byte, []byte, error) {
	return nil, nil, nil
}
func (k *K8sApplierMock) ApplyCreateOrPatch(targetNamespace string, slug string, yamlDoc []byte, dryRun bool, wait bool, annotateSlug bool) ([]byte, []byte, error) {
	return nil, nil, nil
}

func ReturnEmtyListDynamicClientMock(objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
	dynamicClient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), objects...)
	dynamicClient.PrependReactor("list", "*", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, nil
	})
	return dynamicClient
}

func ReturnErrorDynamicClientMock(objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
	dynamicClient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), objects...)
	dynamicClient.PrependReactor("list", "*", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, &kuberneteserrors.StatusError{}
	})
	return dynamicClient
}

func ReturnDynamicClientMock(objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
	dynamicClient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), objects...)
	dynamicClient.PrependReactor("list", "pods", func(action core.Action) (handled bool, ret runtime.Object, err error) {
		list := &unstructured.UnstructuredList{}
		for _, obj := range objects {
			list.Items = append(list.Items, *obj.(*unstructured.Unstructured))
		}
		return true, list, nil
	})
	return dynamicClient
}
