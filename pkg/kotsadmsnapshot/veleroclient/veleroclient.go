package veleroclient

import (
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	rest "k8s.io/client-go/rest"
)

var _ VeleroClientBuilder = (*Builder)(nil)
var _ VeleroClientBuilder = (*MockBuilder)(nil)

// VeleroClientBuilder interface is used as an abstraction to get a velero client. Useful to mock the client in tests.
type VeleroClientBuilder interface {
	GetVeleroClient(*rest.Config) (veleroclientv1.VeleroV1Interface, error)
}

// Builder is the default implementation of VeleroClientBuilder. It returns a regular velero v1 client.
type Builder struct{}

// GetVeleroClient returns a regular velero client.
func (b *Builder) GetVeleroClient(cfg *rest.Config) (veleroclientv1.VeleroV1Interface, error) {
	return veleroclientv1.NewForConfig(cfg)
}

// MockBuilder is a mock implementation of VeleroClientBuilder. It returns the client that was set in the struct allowing
// you to set a fakeClient for example.

type MockBuilder struct {
	Client veleroclientv1.VeleroV1Interface
	err    error
}

// GetVeleroClient returns the client that was set in the struct.
func (b *MockBuilder) GetVeleroClient(cfg *rest.Config) (veleroclientv1.VeleroV1Interface, error) {
	return b.Client, b.err
}

var clientBuilder VeleroClientBuilder

func GetBuilder() VeleroClientBuilder {
	return clientBuilder
}

func SetBuilder(builder VeleroClientBuilder) {
	clientBuilder = builder
}

func init() {
	SetBuilder(&Builder{})
}
