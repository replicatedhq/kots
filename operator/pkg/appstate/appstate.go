package appstate

import (
	"context"
	"log"
	"time"

	"github.com/replicatedhq/kotsadm/operator/pkg/appstate/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Monitor struct {
	clientset       kubernetes.Interface
	targetNamespace string
	appInformersCh  chan appInformer
	appStatusCh     chan types.AppStatus
	cancel          context.CancelFunc
}

type EventHandler interface {
	ObjectCreated(obj interface{})
	ObjectUpdated(obj interface{})
	ObjectDeleted(obj interface{})
}

type appInformer struct {
	appID     string
	informers []types.StatusInformer
}

func NewMonitor(clientset kubernetes.Interface, targetNamespace string) *Monitor {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Monitor{
		clientset:       clientset,
		targetNamespace: targetNamespace,
		appInformersCh:  make(chan appInformer),
		appStatusCh:     make(chan types.AppStatus),
		cancel:          cancel,
	}
	go m.run(ctx)
	return m
}

func (m *Monitor) Shutdown() {
	m.cancel()
}

func (m *Monitor) Apply(appID string, informers []types.StatusInformer) {
	m.appInformersCh <- struct {
		appID     string
		informers []types.StatusInformer
	}{
		appID:     appID,
		informers: informers,
	}
}

func (m *Monitor) AppStatusChan() <-chan types.AppStatus {
	return m.appStatusCh
}

func (m *Monitor) run(ctx context.Context) {
	log.Println("Starting monitor loop")

	defer close(m.appStatusCh)

	appMonitors := make(map[string]*AppMonitor)
	defer func() {
		for _, appMonitor := range appMonitors {
			appMonitor.Shutdown()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case appInformer := <-m.appInformersCh:
			appMonitor, ok := appMonitors[appInformer.appID]
			if !ok {
				appMonitor = NewAppMonitor(m.clientset, m.targetNamespace, appInformer.appID)
				go func() {
					for appStatus := range appMonitor.AppStatusChan() {
						m.appStatusCh <- appStatus
					}
				}()
				appMonitors[appInformer.appID] = appMonitor
			}
			appMonitor.Apply(appInformer.informers)
		}
	}
}

type AppMonitor struct {
	clientset       kubernetes.Interface
	targetNamespace string
	appID           string
	informersCh     chan []types.StatusInformer
	appStatusCh     chan types.AppStatus
	cancel          context.CancelFunc
}

func NewAppMonitor(clientset kubernetes.Interface, targetNamespace, appID string) *AppMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	m := &AppMonitor{
		appID:           appID,
		clientset:       clientset,
		targetNamespace: targetNamespace,
		informersCh:     make(chan []types.StatusInformer),
		appStatusCh:     make(chan types.AppStatus),
		cancel:          cancel,
	}
	go m.run(ctx)
	return m
}

func (m *AppMonitor) Shutdown() {
	m.cancel()
}

func (m *AppMonitor) Apply(informers []types.StatusInformer) {
	m.informersCh <- informers
}

func (m *AppMonitor) AppStatusChan() <-chan types.AppStatus {
	return m.appStatusCh
}

func (m *AppMonitor) run(ctx context.Context) {
	log.Println("Starting app monitor loop")

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

			log.Println("App monitor got new informers")

			ctx, cancel := context.WithCancel(ctx)
			prevCancel = cancel
			m.runInformers(ctx, informers)
		}
	}
}

func (m *AppMonitor) runInformers(ctx context.Context, informers []types.StatusInformer) {
	// TODO: informers only work for the target namespace
	// add support for additional namespaces

	informers = normalizeStatusInformers(informers, m.targetNamespace)

	log.Printf("Running informers: %#v", informers)

	appStatus := types.AppStatus{
		AppID:          m.appID,
		ResourceStates: buildResourceStatesFromStatusInformers(informers),
		UpdatedAt:      time.Now(),
	}
	m.appStatusCh <- appStatus // reset last app status

	resourceStateCh := make(chan types.ResourceState)
	go runDeploymentController(ctx, m.clientset, m.targetNamespace, informers, resourceStateCh)
	go runIngressController(ctx, m.clientset, m.targetNamespace, informers, resourceStateCh)
	go runPersistentVolumeClaimController(ctx, m.clientset, m.targetNamespace, informers, resourceStateCh)
	go runServiceController(ctx, m.clientset, m.targetNamespace, informers, resourceStateCh)
	go runStatefulSetController(ctx, m.clientset, m.targetNamespace, informers, resourceStateCh)

	go func() {
		defer close(resourceStateCh)
		for {
			select {
			case <-ctx.Done():
				return
			case resourceState := <-resourceStateCh:
				appStatus.ResourceStates, _ = resourceStatesApplyNew(appStatus.ResourceStates, informers, resourceState)
				appStatus.UpdatedAt = time.Now() // TODO: this should come from the informer
				m.appStatusCh <- appStatus
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
