package appstate

import (
	"context"
	"log"
	"time"

	"github.com/replicatedhq/kots/pkg/appstate/types"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	DaemonSetResourceKind    = "daemonset"
	DaemonSetPodVersionLabel = "controller-revision-hash"
	DaemonSetOwnerKind       = "DaemonSet"
)

type daemonSetEventHandler struct {
	informers       []types.StatusInformer
	resourceStateCh chan<- types.ResourceState
	clientset       kubernetes.Interface
	targetNamespace string
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
		clientset:       clientset,
		targetNamespace: targetNamespace,
	}

	runInformer(ctx, informer, eventHandler)
}

func (h *daemonSetEventHandler) ObjectCreated(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}

	h.resourceStateCh <- makeDaemonSetResourceState(r, CalculateDaemonSetState(h.clientset, h.targetNamespace, r))
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

	h.resourceStateCh <- makeDaemonSetResourceState(r, CalculateDaemonSetState(h.clientset, h.targetNamespace, r))
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

// The pods in a daemonset can be identified by the match label set in the daemonset and the
// "controller-revision-hash" can be used to determine if they are all the in the same daemonset
// version.
func CalculateDaemonSetState(clientset kubernetes.Interface, targetNamespace string, r *appsv1.DaemonSet) types.State {
	if r == nil {
		return types.StateUnavailable
	}

	if r.Status.ObservedGeneration != r.ObjectMeta.Generation {
		return types.StateUpdating
	}

	listOptions := metav1.ListOptions{LabelSelector: labels.SelectorFromSet(r.Spec.Selector.MatchLabels).String()}

	pods, err := clientset.CoreV1().Pods(targetNamespace).List(context.TODO(), listOptions)
	if err != nil {
		log.Printf("failed to get daemonset pod list: %s", err)
		return types.StateUnavailable
	}

	// If the pod version labels are not all the same, then the daemonset is updating.
	currentVersion := ""
	for _, pod := range pods.Items {
		validOwner := false
		for _, owner := range pod.ObjectMeta.OwnerReferences {
			if owner.Kind == DaemonSetOwnerKind && owner.Name == r.ObjectMeta.Name {
				validOwner = true
				break
			}
		}

		if !validOwner {
			log.Printf("skipping pod %s due to invalid owner references for daemonset %s", pod.ObjectMeta.Name, r.ObjectMeta.Name)
			continue
		}

		version, exists := pod.Labels[DaemonSetPodVersionLabel]
		if !exists {
			log.Printf("failed to find %s label for pod %s", DaemonSetPodVersionLabel, pod.Name)
			return types.StateUnavailable
		}

		if len(currentVersion) == 0 {
			currentVersion = version
		} else if version != currentVersion {
			return types.StateUpdating
		}
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

func makeDaemonSetResourceState(r *appsv1.DaemonSet, state types.State) types.ResourceState {
	return types.ResourceState{
		Kind:      DaemonSetResourceKind,
		Name:      r.Name,
		Namespace: r.Namespace,
		State:     state,
	}
}
