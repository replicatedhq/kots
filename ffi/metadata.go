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
func ReadMetadata(socket, namespace string) {
	go func() {
		var ffiResult *FFIResult

		statusClient, err := connectToStatusServer(socket)
		if err != nil {
			fmt.Printf("failed to connect to status server: %s\n", err)
			return
		}
		defer func() {
			statusClient.end(ffiResult)
		}()

		cfg, err := config.GetConfig()
		if err != nil {
			fmt.Printf("error getting kubernetes config: %s\n", err.Error())
			ffiResult = NewFFIResult(0).WithData(upstream.DefaultMetadata)
			return
		}

		clientset, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			fmt.Printf("error creating kubernetes clientset: %s\n", err.Error())
			ffiResult = NewFFIResult(0).WithData(upstream.DefaultMetadata)
			return
		}

		configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get("kotsadm-application-metadata", metav1.GetOptions{})
		if err != nil {
			if kuberneteserrors.IsNotFound(err) {
				ffiResult = NewFFIResult(0).WithData(upstream.DefaultMetadata)
				return
			}

			fmt.Printf("error reading branding: %s\n", err.Error())
			ffiResult = NewFFIResult(0).WithData(upstream.DefaultMetadata)
			return
		}

		data, ok := configMap.Data["application.yaml"]
		if !ok {
			fmt.Printf("metdata did not contain required key: %#v\n", configMap.Data)
			ffiResult = NewFFIResult(0).WithData(upstream.DefaultMetadata)
			return
		}

		ffiResult = NewFFIResult(0).WithData(data)
	}()
}

//export RemoveMetadata
func RemoveMetadata(socket, namespace string) {
	go func() {
		var ffiResult *FFIResult

		statusClient, err := connectToStatusServer(socket)
		if err != nil {
			fmt.Printf("failed to connect to status server: %s\n", err)
			return
		}
		defer func() {
			statusClient.end(ffiResult)
		}()

		cfg, err := config.GetConfig()
		if err != nil {
			fmt.Printf("error getting kubernetes config: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		clientset, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			fmt.Printf("error creating kubernetes clientset: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		_, err = clientset.CoreV1().ConfigMaps(namespace).Get("kotsadm-application-metadata", metav1.GetOptions{})
		if err != nil {
			if kuberneteserrors.IsNotFound(err) {
				ffiResult = NewFFIResult(0)
				return
			}

			fmt.Printf("error reading branding: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		err = clientset.CoreV1().ConfigMaps(namespace).Delete("kotsadm-application-metadata", &metav1.DeleteOptions{})
		if err != nil {
			fmt.Printf("error deleting metadata: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		ffiResult = NewFFIResult(0)
	}()
}
