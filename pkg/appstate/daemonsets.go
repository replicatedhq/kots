package appstate

import (
	"context"
	"time"

	"github.com/replicatedhq/kots/pkg/appstate/types"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const DaemonSetResourceKind = "daemonset"

var lastSeenGeneration int64 = -1

type daemonSetEventHandler struct {
	informers       []types.StatusInformer
	resourceStateCh chan<- types.ResourceState
}

func init() {
	registerResourceKindNames(DaemonSetResourceKind, "daemonsets", "ds")
}

func runDaemonSetController(ctx context.Context, clientset kubernetes.Interface,
	targetNamespace string, informers []types.StatusInformer, resourceStateCh chan<- types.ResourceState,
) {
	listwatch := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return clientset.AppsV1().DaemonSets(targetNamespace).List(context.TODO(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return clientset.AppsV1().DaemonSets(targetNamespace).Watch(context.TODO(), options)
		},
	}

	informer := cache.NewSharedInformer(listwatch, &appsv1.DaemonSet{}, time.Minute)

	eventHandler := &daemonSetEventHandler{
		informers:       filterStatusInformersByResourceKind(informers, DaemonSetResourceKind),
		resourceStateCh: resourceStateCh,
	}

	runInformer(ctx, informer, eventHandler)
}

func (h *daemonSetEventHandler) ObjectCreated(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}

	h.resourceStateCh <- makeDaemonSetResourceState(r, calculateDaemonSetState(r))
}

func (h *daemonSetEventHandler) ObjectDeleted(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}

	h.resourceStateCh <- makeDaemonSetResourceState(r, types.StateMissing)
}

func (h *daemonSetEventHandler) ObjectUpdated(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}

	h.resourceStateCh <- makeDaemonSetResourceState(r, calculateDaemonSetState(r))
}

func (h *daemonSetEventHandler) getInformer(r *appsv1.DaemonSet) (types.StatusInformer, bool) {
	if r != nil {
		for _, informer := range h.informers {
			if r.Namespace == informer.Namespace && r.Name == informer.Name {
				return informer, true
			}
		}
	}

	return types.StatusInformer{}, false
}

func (h *daemonSetEventHandler) cast(obj interface{}) *appsv1.DaemonSet {
	r, _ := obj.(*appsv1.DaemonSet)
	return r
}

func makeDaemonSetResourceState(r *appsv1.DaemonSet, state types.State) types.ResourceState {
	return types.ResourceState{
		Kind:      DaemonSetResourceKind,
		Name:      r.Name,
		Namespace: r.Namespace,
		State:     state,
	}
}

func calculateDaemonSetState(r *appsv1.DaemonSet) types.State {
	if r == nil {
		return types.StateUnavailable
	}

	if r.Status.ObservedGeneration > lastSeenGeneration {
		if r.Status.UpdatedNumberScheduled < r.Status.DesiredNumberScheduled {
			return types.StateUpdating
		}

		if r.Status.NumberAvailable < r.Status.DesiredNumberScheduled {
			return types.StateUpdating
		}

		lastSeenGeneration = r.Generation
		return types.StateReady
	}

	if r.Status.NumberUnavailable > 0 {
		return types.StateDegraded
	}

	if r.Status.NumberMisscheduled > 0 {
		return types.StateDegraded
	}

	if r.Status.CurrentNumberScheduled != r.Status.DesiredNumberScheduled {
		return types.StateDegraded
	}

	if r.Status.NumberReady >= r.Status.DesiredNumberScheduled {
		return types.StateReady
	}

	if r.Status.NumberReady > 0 {
		return types.StateDegraded
	}

	return types.StateUnavailable
}
