package appstate

import (
	"context"

	"github.com/replicatedhq/kotsadm/operator/pkg/appstate/types"
	appsv1 "k8s.io/api/apps/v1"
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

func runDeploymentController(
	ctx context.Context, clientset kubernetes.Interface, targetNamespace string,
	informers []types.StatusInformer, resourceStateCh chan<- types.ResourceState,
) {
	listwatch := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return clientset.AppsV1().Deployments(targetNamespace).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return clientset.AppsV1().Deployments(targetNamespace).Watch(options)
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
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}
	h.resourceStateCh <- makeDeploymentResourceState(r, calculateDeploymentState(r))
}

func (h *deploymentEventHandler) ObjectUpdated(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}
	h.resourceStateCh <- makeDeploymentResourceState(r, calculateDeploymentState(r))
}

func (h *deploymentEventHandler) ObjectDeleted(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}
	h.resourceStateCh <- makeDeploymentResourceState(r, types.StateMissing)
}

func (h *deploymentEventHandler) cast(obj interface{}) *appsv1.Deployment {
	r, _ := obj.(*appsv1.Deployment)
	return r
}

func (h *deploymentEventHandler) getInformer(r *appsv1.Deployment) (types.StatusInformer, bool) {
	if r != nil {
		for _, informer := range h.informers {
			if r.Namespace == informer.Namespace && r.Name == informer.Name {
				return informer, true
			}
		}
	}
	return types.StatusInformer{}, false
}

func makeDeploymentResourceState(r *appsv1.Deployment, state types.State) types.ResourceState {
	return types.ResourceState{
		Kind:      DeploymentResourceKind,
		Name:      r.Name,
		Namespace: r.Namespace,
		State:     state,
	}
}

func calculateDeploymentState(r *appsv1.Deployment) types.State {
	// https://github.com/kubernetes/kubernetes/blob/badcd4af3f592376ce891b7c1b7a43ed6a18a348/pkg/printers/internalversion/printers.go#L1652
	var desiredReplicas int32
	if r.Spec.Replicas == nil {
		desiredReplicas = 1
	} else {
		desiredReplicas = *r.Spec.Replicas
	}
	if desiredReplicas == 0 {
		// TODO: what to do here?
	}
	if r.Status.ReadyReplicas >= desiredReplicas {
		return types.StateReady
	}
	if r.Status.ReadyReplicas > 0 {
		return types.StateDegraded
	}
	return types.StateUnavailable
}
