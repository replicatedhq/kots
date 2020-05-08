package appstate

import (
	"context"
	"time"

	"github.com/replicatedhq/kotsadm/operator/pkg/appstate/types"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
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
			return clientset.ExtensionsV1beta1().Ingresses(targetNamespace).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return clientset.ExtensionsV1beta1().Ingresses(targetNamespace).Watch(options)
		},
	}
	informer := cache.NewSharedInformer(
		listwatch,
		&extensions.Ingress{},
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
	h.resourceStateCh <- makeIngressResourceState(r, calculateIngressState(h.clientset, r))
}

func (h *ingressEventHandler) ObjectUpdated(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}
	h.resourceStateCh <- makeIngressResourceState(r, calculateIngressState(h.clientset, r))
}

func (h *ingressEventHandler) ObjectDeleted(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}
	h.resourceStateCh <- makeIngressResourceState(r, types.StateMissing)
}

func (h *ingressEventHandler) cast(obj interface{}) *extensions.Ingress {
	r, _ := obj.(*extensions.Ingress)
	return r
}

func (h *ingressEventHandler) getInformer(r *extensions.Ingress) (types.StatusInformer, bool) {
	if r != nil {
		for _, informer := range h.informers {
			if r.Namespace == informer.Namespace && r.Name == informer.Name {
				return informer, true
			}
		}
	}
	return types.StatusInformer{}, false
}

func makeIngressResourceState(r *extensions.Ingress, state types.State) types.ResourceState {
	return types.ResourceState{
		Kind:      IngressResourceKind,
		Name:      r.Name,
		Namespace: r.Namespace,
		State:     state,
	}
}

func calculateIngressState(clientset kubernetes.Interface, r *extensions.Ingress) types.State {
	var states []types.State
	// https://github.com/kubernetes/kubectl/blob/6b77b0790ab40d2a692ad80e9e4c962e784bb9b8/pkg/describe/versioned/describe.go#L2367
	backend := r.Spec.Backend
	ns := r.Namespace
	if backend == nil {
		// Ingresses that don't specify a default backend inherit the
		// default backend in the kube-system namespace.
		backend = &extensions.IngressBackend{
			ServiceName: "default-http-backend",
			ServicePort: intstr.IntOrString{Type: intstr.Int, IntVal: 80},
		}
		ns = metav1.NamespaceSystem
	}
	states = append(states, ingressGetStateFromBackend(clientset, ns, *backend))
	for _, rules := range r.Spec.Rules {
		for _, path := range rules.HTTP.Paths {
			states = append(states, ingressGetStateFromBackend(clientset, r.Namespace, path.Backend))
		}
	}
	// https://github.com/kubernetes/kubernetes/blob/badcd4af3f592376ce891b7c1b7a43ed6a18a348/pkg/printers/internalversion/printers.go#L1067
	states = append(states, ingressGetStateFromExternalIP(r))
	return types.MinState(states...)
}

func ingressGetStateFromBackend(clientset kubernetes.Interface, namespace string, backend extensions.IngressBackend) (minState types.State) {
	service, _ := clientset.CoreV1().Services(namespace).Get(backend.ServiceName, metav1.GetOptions{})
	if service == nil {
		return types.StateUnavailable
	}
	return serviceGetStateFromEndpoints(clientset, service)
}

func ingressGetStateFromExternalIP(ing *extensions.Ingress) types.State {
	lbIps := loadBalancerStatusIPs(ing.Status.LoadBalancer)
	if len(lbIps) > 0 {
		return types.StateReady
	}
	return types.StateUnavailable
}
