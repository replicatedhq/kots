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
	var i interface{} = &h
	r, _ := i.(*appsv1.DaemonSet)

	if _, ok := h.getInformer(r); !ok {
		return
	}

	h.resourceStateCh <- makeDaemonSetResourceState(r, calculateDaemonSetState(r))
}

func (h *daemonSetEventHandler) ObjectDeleted(obj interface{}) {
	var i interface{} = &h
	r, _ := i.(*appsv1.DaemonSet)

	if _, ok := h.getInformer(r); !ok {
		return
	}

	h.resourceStateCh <- makeDaemonSetResourceState(r, types.StateMissing)
}

func (h *daemonSetEventHandler) ObjectUpdated(obj interface{}) {
	var i interface{} = &h
	r, _ := i.(*appsv1.DaemonSet)

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

func makeDaemonSetResourceState(r *appsv1.DaemonSet, state types.State) types.ResourceState {
	return types.ResourceState{
		Kind:      StatefulSetResourceKind,
		Name:      r.Name,
		Namespace: r.Namespace,
		State:     state,
	}
}

func calculateDaemonSetState(r *appsv1.DaemonSet) types.State {
	if r == nil {
		return types.StateUnavailable
	}

	if r.Status.ObservedGeneration != r.ObjectMeta.Generation {
		return types.StateUpdating
	}

	if r.Status.NumberReady >= r.Status.DesiredNumberScheduled {
		return types.StateReady
	} else {
		return types.StateDegraded
	}
}
