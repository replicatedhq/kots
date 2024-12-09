package k8sclient

import (
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
)

var _ K8sClientsetBuilder = (*Builder)(nil)
var _ K8sClientsetBuilder = (*MockBuilder)(nil)

// K8sClientsetBuilder interface is used as an abstraction to get a k8s clientset. Useful to mock the client in tests.
type K8sClientsetBuilder interface {
	GetClientset(*rest.Config) (kubernetes.Interface, error)
}

// Builder is the default implementation of K8sClientsetBuilder. It returns a regular k8s clientset.
type Builder struct{}

// GetClientset returns a regular k8s client.
func (b *Builder) GetClientset(cfg *rest.Config) (kubernetes.Interface, error) {
	return kubernetes.NewForConfig(cfg)
}

// MockBuilder is a mock implementation of K8sClientsetBuilder. It returns the client that was set in the struct allowing
// you to set a fakeClient for example.

type MockBuilder struct {
	Client kubernetes.Interface
	err    error
}

// GetClientset returns the client that was set in the struct.
func (b *MockBuilder) GetClientset(cfg *rest.Config) (kubernetes.Interface, error) {
	return b.Client, b.err
}

var clientBuilder K8sClientsetBuilder

func GetBuilder() K8sClientsetBuilder {
	return clientBuilder
}

func SetBuilder(builder K8sClientsetBuilder) {
	clientBuilder = builder
}

func init() {
	SetBuilder(&Builder{})
}
