package main

import "C"

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/upstream"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

//export ReadMetadata
func ReadMetadata(namespace string) *C.char {
	cfg, err := config.GetConfig()
	if err != nil {
		fmt.Printf("error getting kubernetes config: %s\n", err.Error())
		return C.CString(upstream.DefaultMetadata)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fmt.Printf("error creating kubernetes clientset: %s\n", err.Error())
		return C.CString(upstream.DefaultMetadata)
	}

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get("kotsadm-application-metadata", metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return C.CString(upstream.DefaultMetadata)
		}

		fmt.Printf("error reading branding: %s\n", err.Error())
		return C.CString(upstream.DefaultMetadata)
	}

	data, ok := configMap.Data["application.yaml"]
	if !ok {
		fmt.Printf("metdata did not contain required key: %#v\n", configMap.Data)
		return C.CString(upstream.DefaultMetadata)
	}

	return C.CString(data)
}

//export RemoveMetadata
func RemoveMetadata(namespace string) int {
	cfg, err := config.GetConfig()
	if err != nil {
		fmt.Printf("error getting kubernetes config: %s\n", err.Error())
		return -1
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fmt.Printf("error creating kubernetes clientset: %s\n", err.Error())
		return -1
	}

	_, err = clientset.CoreV1().ConfigMaps(namespace).Get("kotsadm-application-metadata", metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return 0
		}

		fmt.Printf("error reading branding: %s\n", err.Error())
		return -1
	}

	err = clientset.CoreV1().ConfigMaps(namespace).Delete("kotsadm-application-metadata", &metav1.DeleteOptions{})
	if err != nil {
		fmt.Printf("error deleting metadata: %s\n", err.Error())
		return -1
	}

	return 0
}
