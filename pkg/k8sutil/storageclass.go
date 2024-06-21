package k8sutil

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetStorageClassName returns the name of the storage class in the cluster.
// If there is more than one storage class or an error occurs, nil is returned so that the default storage class is used.
// This is needed for EKS 1.30+ clusters where the default is that no storage class is default.
func GetStorageClassName() *string {
	clientset, err := GetClientset()
	if err != nil {
		return nil
	}
	storageClasses, err := clientset.StorageV1().StorageClasses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	if len(storageClasses.Items) != 1 {
		return nil
	}
	return &storageClasses.Items[0].Name
}
