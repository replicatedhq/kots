package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type OverlySimpleGVKWithName struct {
	APIVersion string               `yaml:"apiVersion"`
	Kind       string               `yaml:"kind"`
	Metadata   OverlySimpleMetadata `yaml:"metadata"`
}

type OverlySimpleMetadata struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels,omitempty"`
}

func GetGVKWithNameAndNs(content []byte, baseNS string) (string, OverlySimpleGVKWithName) {
	o := OverlySimpleGVKWithName{}

	if err := yaml.Unmarshal(content, &o); err != nil {
		return "", o
	}

	namespace := baseNS
	if o.Metadata.Namespace != "" {
		namespace = o.Metadata.Namespace
	}

	return fmt.Sprintf("%s-%s-%s-%s", o.APIVersion, o.Kind, o.Metadata.Name, namespace), o
}

func IsCRD(content []byte) bool {
	o := OverlySimpleGVKWithName{}

	if err := yaml.Unmarshal(content, &o); err != nil {
		return false
	}

	if o.Kind == "CustomResourceDefinition" {
		return strings.HasPrefix(o.APIVersion, "apiextensions.k8s.io/")
	}

	return false
}

func IsNamespace(content []byte) bool {
	o := OverlySimpleGVKWithName{}

	if err := yaml.Unmarshal(content, &o); err != nil {
		return false
	}

	return o.APIVersion == "v1" && o.Kind == "Namespace"
}

func findPodsByOwner(name string, namespace string, gvk *k8sschema.GroupVersionKind) ([]*corev1.Pod, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get client set")
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list pods")
	}

	matchingPods := make([]*corev1.Pod, 0)
	for _, pod := range pods.Items {
		for _, owner := range pod.OwnerReferences {
			if owner.Name == name && owner.Kind == gvk.Kind && owner.APIVersion == gvk.GroupVersion().String() {
				matchingPods = append(matchingPods, pod.DeepCopy())
			}
		}
	}

	return matchingPods, nil
}

func findPodByName(name string, namespace string) (*corev1.Pod, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get client set")
	}

	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pod")
	}

	return pod, nil
}
