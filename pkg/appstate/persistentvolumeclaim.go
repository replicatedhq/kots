package appstate

import (
	"context"
	"time"

	"github.com/replicatedhq/kots/pkg/appstate/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	PersistentVolumeClaimResourceKind = "persistentvolumeclaim"
)

func init() {
	registerResourceKindNames(PersistentVolumeClaimResourceKind, "persistentvolumeclaims", "pvc")
}

func runPersistentVolumeClaimController(
	ctx context.Context, clientset kubernetes.Interface, targetNamespace string,
	informers []types.StatusInformer, resourceStateCh chan<- types.ResourceState,
) {
	listwatch := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return clientset.CoreV1().PersistentVolumeClaims(targetNamespace).List(context.TODO(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return clientset.CoreV1().PersistentVolumeClaims(targetNamespace).Watch(context.TODO(), options)
		},
	}
	informer := cache.NewSharedInformer(
		listwatch,
		&corev1.PersistentVolumeClaim{},
		time.Minute,
	)

	eventHandler := NewPersistentVolumeClaimEventHandler(
		filterStatusInformersByResourceKind(informers, PersistentVolumeClaimResourceKind),
		resourceStateCh,
	)

	runInformer(ctx, informer, eventHandler)
	return
}

type persistentVolumeClaimEventHandler struct {
	informers       []types.StatusInformer
	resourceStateCh chan<- types.ResourceState
}

func NewPersistentVolumeClaimEventHandler(informers []types.StatusInformer, resourceStateCh chan<- types.ResourceState) *persistentVolumeClaimEventHandler {
	return &persistentVolumeClaimEventHandler{
		informers:       informers,
		resourceStateCh: resourceStateCh,
	}
}

func (h *persistentVolumeClaimEventHandler) ObjectCreated(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}
	h.resourceStateCh <- makePersistentVolumeClaimResourceState(r, CalculatePersistentVolumeClaimState(r))
}

func (h *persistentVolumeClaimEventHandler) ObjectUpdated(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}
	h.resourceStateCh <- makePersistentVolumeClaimResourceState(r, CalculatePersistentVolumeClaimState(r))
}

func (h *persistentVolumeClaimEventHandler) ObjectDeleted(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}
	h.resourceStateCh <- makePersistentVolumeClaimResourceState(r, types.StateMissing)
}

func (h *persistentVolumeClaimEventHandler) cast(obj interface{}) *corev1.PersistentVolumeClaim {
	r, _ := obj.(*corev1.PersistentVolumeClaim)
	return r
}

func (h *persistentVolumeClaimEventHandler) getInformer(r *corev1.PersistentVolumeClaim) (types.StatusInformer, bool) {
	if r != nil {
		for _, informer := range h.informers {
			if r.Namespace == informer.Namespace && r.Name == informer.Name {
				return informer, true
			}
		}
	}
	return types.StatusInformer{}, false
}

func makePersistentVolumeClaimResourceState(r *corev1.PersistentVolumeClaim, state types.State) types.ResourceState {
	return types.ResourceState{
		Kind:      PersistentVolumeClaimResourceKind,
		Name:      r.Name,
		Namespace: r.Namespace,
		State:     state,
	}
}

func CalculatePersistentVolumeClaimState(r *corev1.PersistentVolumeClaim) types.State {
	// https://github.com/kubernetes/kubernetes/blob/badcd4af3f592376ce891b7c1b7a43ed6a18a348/pkg/printers/internalversion/printers.go#L1403
	switch r.Status.Phase {
	case corev1.ClaimPending, corev1.ClaimLost:
		return types.StateUnavailable
	case corev1.ClaimBound:
		return types.StateReady
	default:
		// I'm not sure what state to return here
		return types.StateUnavailable
	}
}
