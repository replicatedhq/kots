package kubeclient

import (
	"context"

	"github.com/replicatedhq/kots/pkg/k8sutil"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ KubeClientBuilder = (*Builder)(nil)
var _ KubeClientBuilder = (*MockBuilder)(nil)

// KubeClientBuilder interface is used as an abstraction to get a kube client. Useful to mock the client in tests.
type KubeClientBuilder interface {
	GetKubeClient(ctx context.Context) (kbclient.Client, error)
}

// Builder is the default implementation of KubeClientBuilder. It returns a regular kube client.
type Builder struct{}

// GetKubeClient returns a regular kube client.
func (b *Builder) GetKubeClient(ctx context.Context) (kbclient.Client, error) {
	return k8sutil.GetKubeClient(ctx)
}

// MockBuilder is a mock implementation of KubeClientBuilder. It returns the client that was set in the struct allowing
// you to set a fakeClient for example.
type MockBuilder struct {
	Client kbclient.Client
}

// GetKubeClient returns the client that was set in the struct.
func (b *MockBuilder) GetKubeClient(ctx context.Context) (kbclient.Client, error) {
	return b.Client, nil
}
