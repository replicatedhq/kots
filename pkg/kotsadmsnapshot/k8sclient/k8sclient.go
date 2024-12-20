package k8sclient

import (
	"context"

	"github.com/replicatedhq/kots/pkg/k8sutil"
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ K8sClientsBuilder = (*Builder)(nil)
var _ K8sClientsBuilder = (*MockBuilder)(nil)

// K8sClientsBuilder interface is used as an abstraction to get a kubernetes go client clientset.
// Or a controller runtime client. Useful to mock the client in tests.
type K8sClientsBuilder interface {
	GetClientset(*rest.Config) (kubernetes.Interface, error)
	GetKubeClient(ctx context.Context) (kbclient.Client, error)
	GetVeleroKubeClient(ctx context.Context) (kbclient.Client, error)
}

// Builder is the default implementation of K8sClientsetBuilder. It returns a regular go client clientset or a regular controller runtime client.
type Builder struct{}

// GetClientset returns a regular go client for the given config.
func (b *Builder) GetClientset(cfg *rest.Config) (kubernetes.Interface, error) {
	return kubernetes.NewForConfig(cfg)
}

// GetKubeClient returns a regular controller runtime client based on the default cluster config provided.
func (b *Builder) GetKubeClient(ctx context.Context) (kbclient.Client, error) {
	return k8sutil.GetKubeClient(ctx)
}

// GetVeleroKubeClient returns a controller runtime client with the velero API scheme based on the default cluster config provided.
func (b *Builder) GetVeleroKubeClient(ctx context.Context) (kbclient.Client, error) {
	return k8sutil.GetVeleroKubeClient(ctx)
}

// MockBuilder is a mock implementation of K8sClientsetBuilder. It returns the client and clientset that was set in the struct allowing
// you to set a fake clients for example.

type MockBuilder struct {
	Clientset  kubernetes.Interface
	CtrlClient kbclient.Client
	Err        error
}

// GetClientset returns the clientset that was set in the struct.
func (b *MockBuilder) GetClientset(cfg *rest.Config) (kubernetes.Interface, error) {
	return b.Clientset, b.Err
}

// GetKubeClient returns the controller runtime client set in the struct.
func (b *MockBuilder) GetKubeClient(ctx context.Context) (kbclient.Client, error) {
	return b.CtrlClient, b.Err
}

// GetVeleroKubeClient returns the controller runtime client set in the struct.
func (b *MockBuilder) GetVeleroKubeClient(ctx context.Context) (kbclient.Client, error) {
	return b.CtrlClient, b.Err
}

var clientBuilder K8sClientsBuilder

func GetBuilder() K8sClientsBuilder {
	return clientBuilder
}

func SetBuilder(builder K8sClientsBuilder) {
	clientBuilder = builder
}

func init() {
	SetBuilder(&Builder{})
}
