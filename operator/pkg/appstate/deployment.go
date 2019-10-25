package appstate

import (
	"context"

	"github.com/replicatedhq/kotsadm/operator/pkg/appstate/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	DeploymentResourceKind = "deployment"
)

func init() {
	registerResourceKindNames(DeploymentResourceKind, "deployments", "deploy")
}

func runDeploymentController(ctx context.Context, clientset kubernetes.Interface, informers []types.StatusInformer, resourceStateCh chan<- types.ResourceState) {
	listwatch := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return clientset.AppsV1().Deployments(corev1.NamespaceAll).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return clientset.AppsV1().Deployments(corev1.NamespaceAll).Watch(options)
		},
	}
	informer := cache.NewSharedInformer(
		listwatch,
		&appsv1.Deployment{},
		0, //Skip resync,
	)

	eventHandler := NewDeploymentEventHandler(
		filterStatusInformersByResourceKind(informers, DeploymentResourceKind),
		resourceStateCh,
	)

	runInformer(ctx, informer, eventHandler)
	return
}

type deploymentEventHandler struct {
	informers       []types.StatusInformer
	resourceStateCh chan<- types.ResourceState
}

func NewDeploymentEventHandler(informers []types.StatusInformer, resourceStateCh chan<- types.ResourceState) *deploymentEventHandler {
	return &deploymentEventHandler{
		informers:       informers,
		resourceStateCh: resourceStateCh,
	}
}

func (h *deploymentEventHandler) ObjectCreated(obj interface{}) {
	deploy := h.cast(obj)
	if _, ok := h.getInformer(deploy); !ok {
		return
	}
	h.resourceStateCh <- makeDeploymentResourceState(deploy, calculateDeploymentState(deploy))
}

func (h *deploymentEventHandler) ObjectUpdated(obj interface{}) {
	deploy := h.cast(obj)
	if _, ok := h.getInformer(deploy); !ok {
		return
	}
	h.resourceStateCh <- makeDeploymentResourceState(deploy, calculateDeploymentState(deploy))
}

func (h *deploymentEventHandler) ObjectDeleted(obj interface{}) {
	deploy := h.cast(obj)
	if _, ok := h.getInformer(deploy); !ok {
		return
	}
	h.resourceStateCh <- makeDeploymentResourceState(deploy, types.StateMissing)
}

func (h *deploymentEventHandler) cast(obj interface{}) *appsv1.Deployment {
	deploy, _ := obj.(*appsv1.Deployment)
	return deploy
}

func (h *deploymentEventHandler) getInformer(deploy *appsv1.Deployment) (types.StatusInformer, bool) {
	if deploy != nil {
		for _, informer := range h.informers {
			if deploy.Namespace == informer.Namespace && deploy.Name == informer.Name {
				return informer, true
			}
		}
	}
	return types.StatusInformer{}, false
}

func makeDeploymentResourceState(deploy *appsv1.Deployment, state types.State) types.ResourceState {
	return types.ResourceState{
		Kind:      DeploymentResourceKind,
		Name:      deploy.Name,
		Namespace: deploy.Namespace,
		State:     state,
	}
}

func calculateDeploymentState(deploy *appsv1.Deployment) types.State {
	if deploy.Status.Replicas == 0 {
		// TODO: what to do here?
	}
	if deploy.Status.AvailableReplicas == deploy.Status.Replicas {
		return types.StateReady
	}
	if deploy.Status.AvailableReplicas > 0 {
		return types.StateDegraded
	}
	return types.StateUnavailable
}
