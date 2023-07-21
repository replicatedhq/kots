package appstate

import (
	"context"
	"time"

	"github.com/replicatedhq/kots/pkg/appstate/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	IngressResourceKind = "ingress"
)

func init() {
	registerResourceKindNames(IngressResourceKind, "ingresses", "ing")
}

func runIngressController(
	ctx context.Context, clientset kubernetes.Interface, targetNamespace string,
	informers []types.StatusInformer, resourceStateCh chan<- types.ResourceState,
) {
	listwatch := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return clientset.NetworkingV1().Ingresses(targetNamespace).List(context.TODO(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return clientset.NetworkingV1().Ingresses(targetNamespace).Watch(context.TODO(), options)
		},
	}
	informer := cache.NewSharedInformer(
		listwatch,
		&networkingv1.Ingress{},
		// NOTE: ingresses rely on endpoint and service status as well so unless we add
		// additional informers, we have to resync more frequently.
		10*time.Second,
	)

	eventHandler := NewIngressEventHandler(
		clientset,
		filterStatusInformersByResourceKind(informers, IngressResourceKind),
		resourceStateCh,
	)

	runInformer(ctx, informer, eventHandler)
	return
}

type ingressEventHandler struct {
	clientset       kubernetes.Interface
	informers       []types.StatusInformer
	resourceStateCh chan<- types.ResourceState
}

func NewIngressEventHandler(clientset kubernetes.Interface, informers []types.StatusInformer, resourceStateCh chan<- types.ResourceState) *ingressEventHandler {
	return &ingressEventHandler{
		clientset:       clientset,
		informers:       informers,
		resourceStateCh: resourceStateCh,
	}
}

func (h *ingressEventHandler) ObjectCreated(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}
	h.resourceStateCh <- makeIngressResourceState(r, CalculateIngressState(h.clientset, r))
}

func (h *ingressEventHandler) ObjectUpdated(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}
	h.resourceStateCh <- makeIngressResourceState(r, CalculateIngressState(h.clientset, r))
}

func (h *ingressEventHandler) ObjectDeleted(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}
	h.resourceStateCh <- makeIngressResourceState(r, types.StateMissing)
}

func (h *ingressEventHandler) cast(obj interface{}) *networkingv1.Ingress {
	r, _ := obj.(*networkingv1.Ingress)
	return r
}

func (h *ingressEventHandler) getInformer(r *networkingv1.Ingress) (types.StatusInformer, bool) {
	if r != nil {
		for _, informer := range h.informers {
			if r.Namespace == informer.Namespace && r.Name == informer.Name {
				return informer, true
			}
		}
	}
	return types.StatusInformer{}, false
}

func makeIngressResourceState(r *networkingv1.Ingress, state types.State) types.ResourceState {
	return types.ResourceState{
		Kind:      IngressResourceKind,
		Name:      r.Name,
		Namespace: r.Namespace,
		State:     state,
	}
}

func CalculateIngressState(clientset kubernetes.Interface, r *networkingv1.Ingress) types.State {
	ns := r.Namespace
	backend := r.Spec.DefaultBackend

	k8sMinorVersion, err := k8sutil.GetK8sMinorVersion(clientset)
	if err != nil {
		logger.Errorf("failed to get k8s minor version: %v", err)
	} else if k8sMinorVersion < 22 && backend == nil {
		// https://github.com/kubernetes/kubectl/blob/6b77b0790ab40d2a692ad80e9e4c962e784bb9b8/pkg/describe/versioned/describe.go#L2367
		// Ingresses that don't specify a default backend inherit the default backend in the kube-system namespace.
		// This behavior is applicable to Kubernetes versions prior to 1.22 (i.e. Ingress versions before networking.k8s.io/v1).
		backend = &networkingv1.IngressBackend{
			Service: &networkingv1.IngressServiceBackend{
				Name: "default-http-backend",
				Port: networkingv1.ServiceBackendPort{
					Number: 80,
				},
			},
		}
		ns = metav1.NamespaceSystem
	}

	var states []types.State
	if backend != nil {
		states = append(states, ingressGetStateFromBackend(clientset, ns, *backend))
	}

	for _, rules := range r.Spec.Rules {
		for _, path := range rules.HTTP.Paths {
			states = append(states, ingressGetStateFromBackend(clientset, r.Namespace, path.Backend))
		}
	}
	// https://github.com/kubernetes/kubernetes/blob/badcd4af3f592376ce891b7c1b7a43ed6a18a348/pkg/printers/internalversion/printers.go#L1067
	states = append(states, ingressGetStateFromExternalIP(r))
	return types.MinState(states...)
}

func ingressGetStateFromBackend(clientset kubernetes.Interface, namespace string, backend networkingv1.IngressBackend) (minState types.State) {
	if backend.Service == nil {
		return types.StateUnavailable
	}
	service, _ := clientset.CoreV1().Services(namespace).Get(context.TODO(), backend.Service.Name, metav1.GetOptions{})
	if service == nil {
		return types.StateUnavailable
	}
	return serviceGetStateFromEndpoints(clientset, service)
}

func ingressGetStateFromExternalIP(ing *networkingv1.Ingress) types.State {
	lbIps := loadBalancerStatusIPs(ing.Status.LoadBalancer)
	if len(lbIps) > 0 {
		return types.StateReady
	}
	return types.StateUnavailable
}
