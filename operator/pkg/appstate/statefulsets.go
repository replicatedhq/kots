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
	StatefulSetResourceKind = "statefulset"
)

func init() {
	registerResourceKindNames(StatefulSetResourceKind, "statefulsets", "sts")
}

func runStatefulSetController(ctx context.Context, clientset kubernetes.Interface, informers []types.StatusInformer, resourceStateCh chan<- types.ResourceState) {
	listwatch := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return clientset.AppsV1().StatefulSets(corev1.NamespaceAll).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return clientset.AppsV1().StatefulSets(corev1.NamespaceAll).Watch(options)
		},
	}
	informer := cache.NewSharedInformer(
		listwatch,
		&appsv1.StatefulSet{},
		0, //Skip resync,
	)

	eventHandler := NewStatefulSetEventHandler(
		filterStatusInformersByResourceKind(informers, StatefulSetResourceKind),
		resourceStateCh,
	)

	runInformer(ctx, informer, eventHandler)
	return
}

type statefulSetEventHandler struct {
	informers       []types.StatusInformer
	resourceStateCh chan<- types.ResourceState
}

func NewStatefulSetEventHandler(informers []types.StatusInformer, resourceStateCh chan<- types.ResourceState) *statefulSetEventHandler {
	return &statefulSetEventHandler{
		informers:       informers,
		resourceStateCh: resourceStateCh,
	}
}

func (h *statefulSetEventHandler) ObjectCreated(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}
	h.resourceStateCh <- makeStatefulSetResourceState(r, calculateStatefulSetState(r))
}

func (h *statefulSetEventHandler) ObjectUpdated(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}
	h.resourceStateCh <- makeStatefulSetResourceState(r, calculateStatefulSetState(r))
}

func (h *statefulSetEventHandler) ObjectDeleted(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}
	h.resourceStateCh <- makeStatefulSetResourceState(r, types.StateMissing)
}

func (h *statefulSetEventHandler) cast(obj interface{}) *appsv1.StatefulSet {
	r, _ := obj.(*appsv1.StatefulSet)
	return r
}

func (h *statefulSetEventHandler) getInformer(r *appsv1.StatefulSet) (types.StatusInformer, bool) {
	if r != nil {
		for _, informer := range h.informers {
			if r.Namespace == informer.Namespace && r.Name == informer.Name {
				return informer, true
			}
		}
	}
	return types.StatusInformer{}, false
}

func makeStatefulSetResourceState(r *appsv1.StatefulSet, state types.State) types.ResourceState {
	return types.ResourceState{
		Kind:      StatefulSetResourceKind,
		Name:      r.Name,
		Namespace: r.Namespace,
		State:     state,
	}
}

func calculateStatefulSetState(r *appsv1.StatefulSet) types.State {
	// https://github.com/kubernetes/kubernetes/blob/badcd4af3f592376ce891b7c1b7a43ed6a18a348/pkg/printers/internalversion/printers.go#L1098
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
