package k8sutil

import (
	openapi_v2 "github.com/googleapis/gnostic/openapiv2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

type CachedDiscoveryClient struct {
	clientSet *kubernetes.Clientset
}

func (c CachedDiscoveryClient) RESTClient() restclient.Interface {
	return c.clientSet.RESTClient()
}

func (c CachedDiscoveryClient) ServerGroups() (*metav1.APIGroupList, error) {
	return c.clientSet.ServerGroups()
}

func (c CachedDiscoveryClient) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	return c.clientSet.ServerResourcesForGroupVersion(groupVersion)
}

func (c CachedDiscoveryClient) ServerResources() ([]*metav1.APIResourceList, error) {
	return c.clientSet.ServerResources()
}

func (c CachedDiscoveryClient) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	return c.clientSet.ServerGroupsAndResources()
}

func (c CachedDiscoveryClient) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	return c.clientSet.ServerPreferredResources()
}

func (c CachedDiscoveryClient) ServerPreferredNamespacedResources() ([]*metav1.APIResourceList, error) {
	return c.clientSet.ServerPreferredNamespacedResources()
}

func (c CachedDiscoveryClient) ServerVersion() (*version.Info, error) {
	return c.clientSet.ServerVersion()
}

func (c CachedDiscoveryClient) OpenAPISchema() (*openapi_v2.Document, error) {
	return c.clientSet.OpenAPISchema()
}

func (c CachedDiscoveryClient) Fresh() bool {
	return true
}

func (c CachedDiscoveryClient) Invalidate() {
}
