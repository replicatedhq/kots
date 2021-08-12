package appstate

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/replicatedhq/kots/pkg/appstate/types"
	corev1 "k8s.io/api/core/v1"
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
	sequence  int64
	informers []types.StatusInformer
}

func NewMonitor(clientset kubernetes.Interface, targetNamespace string) *Monitor {
	if targetNamespace == "" {
		targetNamespace = corev1.NamespaceDefault
	}
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

func (m *Monitor) Apply(appID string, sequence int64, informers []types.StatusInformer) {
	m.appInformersCh <- struct {
		appID     string
		sequence  int64
		informers []types.StatusInformer
	}{
		appID:     appID,
		sequence:  sequence,
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
			if !ok || appMonitor.sequence != appInformer.sequence {
				if appMonitor != nil {
					appMonitor.Shutdown()
				}
				appMonitor = NewAppMonitor(m.clientset, m.targetNamespace, appInformer.appID, appInformer.sequence)
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
	sequence        int64
}

func NewAppMonitor(clientset kubernetes.Interface, targetNamespace, appID string, sequence int64) *AppMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	m := &AppMonitor{
		appID:           appID,
		clientset:       clientset,
		targetNamespace: targetNamespace,
		informersCh:     make(chan []types.StatusInformer),
		appStatusCh:     make(chan types.AppStatus),
		cancel:          cancel,
		sequence:        sequence,
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
	defer func() {
		// wrap this in a function to cancel the variable when updated
		prevCancel()
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case informers := <-m.informersCh:
			prevCancel() // cancel previous loop

			log.Println("App monitor got new informers")

			ctx, cancel := context.WithCancel(ctx)
			prevCancel = cancel
			go m.runInformers(ctx, informers)
		}
	}
}

type runControllerFunc func(context.Context, kubernetes.Interface, string, []types.StatusInformer, chan<- types.ResourceState)

func (m *AppMonitor) runInformers(ctx context.Context, informers []types.StatusInformer) {
	informers = normalizeStatusInformers(informers, m.targetNamespace)

	log.Printf("Running informers: %#v", informers)

	appStatus := types.AppStatus{
		AppID:          m.appID,
		ResourceStates: buildResourceStatesFromStatusInformers(informers),
		UpdatedAt:      time.Now(),
		Sequence:       m.sequence,
	}
	m.appStatusCh <- appStatus // reset last app status

	var shutdown sync.WaitGroup
	resourceStateCh := make(chan types.ResourceState)
	defer func() {
		shutdown.Wait()
		close(resourceStateCh)
	}()

	// Collect namespace/kind pairs
	namespaceKinds := make(map[string]map[string][]types.StatusInformer)
	for _, informer := range informers {
		kindsInNs, ok := namespaceKinds[informer.Namespace]
		if !ok {
			kindsInNs = make(map[string][]types.StatusInformer)
		}
		kindsInNs[informer.Kind] = append(kindsInNs[informer.Kind], informer)
		namespaceKinds[informer.Namespace] = kindsInNs
	}

	goRun := func(fn runControllerFunc, namespace string, informers []types.StatusInformer) {
		shutdown.Add(1)
		go func() {
			fn(ctx, m.clientset, namespace, informers, resourceStateCh)
			shutdown.Done()
		}()
	}

	kindImpls := map[string]runControllerFunc{
		DeploymentResourceKind:            runDeploymentController,
		IngressResourceKind:               runIngressController,
		PersistentVolumeClaimResourceKind: runPersistentVolumeClaimController,
		ServiceResourceKind:               runServiceController,
		StatefulSetResourceKind:           runStatefulSetController,
	}
	for namespace, kinds := range namespaceKinds {
		for kind, informers := range kinds {
			if impl, ok := kindImpls[kind]; ok {
				goRun(impl, namespace, informers)
			} else {
				log.Printf("Informer requested for unsupported resource kind %v", kind)
			}
		}
	}

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
