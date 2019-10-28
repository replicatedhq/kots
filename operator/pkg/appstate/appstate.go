package appstate

import (
	"context"
	"log"
	"sync"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/operator/pkg/appstate/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type Monitor struct {
	clientset        kubernetes.Interface
	defaultNamespace string
	informersCh      chan []types.StatusInformer
	appStatusCh      chan types.AppStatus
	cancel           context.CancelFunc

	sync.Mutex
}

type EventHandler interface {
	ObjectCreated(obj interface{})
	ObjectUpdated(obj interface{})
	ObjectDeleted(obj interface{})
}

func NewMonitor(restconfig *rest.Config, defaultNamespace string) (*Monitor, error) {
	clientset, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return nil, errors.Wrap(err, "new kubernetes client")
	}

	ctx, cancel := context.WithCancel(context.Background())
	m := &Monitor{
		clientset:        clientset,
		defaultNamespace: defaultNamespace,
		informersCh:      make(chan []types.StatusInformer),
		appStatusCh:      make(chan types.AppStatus),
		cancel:           cancel,
	}
	go m.run(ctx)
	return m, nil
}

func (m *Monitor) Shutdown() {
	m.cancel()
}

func (m *Monitor) Apply(informers []types.StatusInformer) {
	m.informersCh <- informers
}

func (m *Monitor) AppStatusChan() <-chan types.AppStatus {
	return m.appStatusCh
}

func (m *Monitor) run(ctx context.Context) {
	log.Println("Starting appstate monitor loop")

	defer close(m.informersCh)
	defer close(m.appStatusCh)

	prevCancel := context.CancelFunc(func() {})
	defer prevCancel()

	for {
		select {
		case <-ctx.Done():
			return

		case informers := <-m.informersCh:
			prevCancel() // cancel previous loop

			log.Println("Appstate monitor got new informers")

			ctx, cancel := context.WithCancel(ctx)
			prevCancel = cancel
			m.runInformers(ctx, informers)
		}
	}
}

func (m *Monitor) runInformers(ctx context.Context, informers []types.StatusInformer) {
	informers = normalizeStatusInformers(informers, m.defaultNamespace)

	log.Printf("Running informers: %#v", informers)

	appStatus := buildAppStatusFromStatusInformers(informers)
	var didSendOnce bool

	resourceStateCh := make(chan types.ResourceState)
	go runDeploymentController(ctx, m.clientset, informers, resourceStateCh)

	go func() {
		defer close(resourceStateCh)
		for {
			select {
			case <-ctx.Done():
				return
			case resourceState := <-resourceStateCh:
				if a, didChange := appStatusApplyNewResourceState(appStatus, informers, resourceState); didChange || !didSendOnce {
					appStatus = a
					m.appStatusCh <- appStatus
					didSendOnce = true
				}
			}
		}
	}()
}

func runInformer(ctx context.Context, informer cache.SharedInformer, eventHandler EventHandler) {
	defer utilruntime.HandleCrash()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			eventHandler.ObjectCreated(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			eventHandler.ObjectUpdated(new)
		},
		DeleteFunc: func(obj interface{}) {
			eventHandler.ObjectDeleted(obj)
		},
	})

	informer.Run(ctx.Done())
}
