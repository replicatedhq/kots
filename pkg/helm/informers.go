package helm

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mitchellh/hashstructure"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/appstate"
	appstatetypes "github.com/replicatedhq/kots/pkg/appstate/types"
	identitydeploy "github.com/replicatedhq/kots/pkg/identity/deploy"
	identitytypes "github.com/replicatedhq/kots/pkg/identity/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/util"
	"k8s.io/client-go/kubernetes"
)

var monitorMap map[string]*appstate.Monitor
var monitorMux *sync.Mutex

func initMonitor(clientset kubernetes.Interface, targetNamespace string) {
	if monitorMap == nil {
		monitorMap = make(map[string]*appstate.Monitor)
		monitorMux = new(sync.Mutex)
	}

	monitorMux.Lock()
	if monitorMap[targetNamespace] == nil {
		monitorMap[targetNamespace] = appstate.NewMonitor(clientset, targetNamespace)
	}
	nsMon := monitorMap[targetNamespace]
	monitorMux.Unlock()

	go runAppStateMonitor(nsMon)
}

func getMonitor(namespace string) *appstate.Monitor {
	monitorMux.Lock()
	defer monitorMux.Unlock()
	return monitorMap[namespace]
}

func resumeHelmStatusInformers(appName string) {
	helmApp := GetHelmApp(appName)
	app := &apptypes.App{
		ID:                helmApp.GetID(),
		Slug:              helmApp.GetSlug(),
		Name:              helmApp.Release.Name,
		IsAirgap:          helmApp.GetIsAirgap(),
		CurrentSequence:   helmApp.GetCurrentSequence(),
		UpdatedAt:         &helmApp.CreationTimestamp,
		CreatedAt:         helmApp.CreationTimestamp,
		LastUpdateCheckAt: &helmApp.CreationTimestamp,
		IsConfigurable:    helmApp.IsConfigurable,
	}

	kotsKinds, err := GetKotsKindsFromHelmApp(helmApp)
	if err != nil {
		logger.Errorf("failed to get kots kinds from helm app: %v\n", err)
		return
	}

	settings := registrytypes.RegistrySettings{
		Namespace: util.PodNamespace,
	}
	builder, err := render.NewBuilder(&kotsKinds, settings, app.Slug, helmApp.GetCurrentSequence(), app.IsAirgap, util.PodNamespace)
	if err != nil {
		logger.Errorf("failed to create new builder: %v\n", err)
		return
	}

	// apply the informers for this app since they havent been applied yet
	if err := applyStatusInformers(app, helmApp.GetCurrentSequence(), &kotsKinds, builder, helmApp.Namespace); err != nil {
		logger.Errorf("failed to apply status informers: %v\n", err)
		return
	}
}

func applyStatusInformers(a *apptypes.App, sequence int64, kotsKinds *kotsutil.KotsKinds, builder *template.Builder, namespace string) error {
	renderedInformers := []appstatetypes.StatusInformerString{}

	// render status informers
	for _, informer := range kotsKinds.KotsApplication.Spec.StatusInformers {
		renderedInformer, err := builder.String(informer)
		if err != nil {
			logger.Errorf("failed to render status informer: %v\n", err)
			continue
		}
		if renderedInformer == "" {
			continue
		}
		renderedInformers = append(renderedInformers, appstatetypes.StatusInformerString(renderedInformer))
	}

	if identitydeploy.IsEnabled(kotsKinds.Identity, kotsKinds.IdentityConfig) {
		renderedInformers = append(renderedInformers, appstatetypes.StatusInformerString(fmt.Sprintf("deployment/%s", identitytypes.DeploymentName(a.Slug))))
	}

	if len(renderedInformers) > 0 {
		applyAppInformers(a.ID, sequence, renderedInformers, namespace)
	} else {
		// no informers, set state to ready
		defaultReadyState := appstatetypes.ResourceStates{
			{
				Kind:      "EMPTY",
				Name:      "EMPTY",
				Namespace: "EMPTY",
				State:     appstatetypes.StateReady,
			},
		}

		release := GetHelmApp(a.ID)
		appStatus := appstatetypes.AppStatus{
			AppID:          a.ID,
			ResourceStates: defaultReadyState,
			Sequence:       release.GetCurrentSequence(),
			UpdatedAt:      time.Now(),
		}
		release.Status = appStatus
		release.Status.State = appstatetypes.GetState(defaultReadyState)
	}

	return nil
}

func applyAppInformers(appID string, sequence int64, informerStrings []appstatetypes.StatusInformerString, namespace string) {
	logger.Infof("received an inform event for appID (%s) sequence (%d) with informerStrings (%+v)\n", appID, sequence, informerStrings)

	var informers []appstatetypes.StatusInformer
	for _, str := range informerStrings {
		informer, err := str.Parse()
		if err != nil {
			logger.Infof("failed to parse informer %s: %s", str, err.Error())
			continue // don't stop
		}
		informers = append(informers, informer)
	}
	if len(informers) > 0 {
		if mon := getMonitor(namespace); mon != nil {
			mon.Apply(appID, sequence, informers)
		}
	}
}

func runAppStateMonitor(monitor *appstate.Monitor) error {
	m := map[string]func(f func()){}
	hash := map[string]uint64{}
	var mtx sync.Mutex

	for appStatus := range monitor.AppStatusChan() {
		throttled, ok := m[appStatus.AppID]
		if !ok {
			throttled = util.NewThrottle(time.Second)
			m[appStatus.AppID] = throttled
		}
		throttled(func() {
			mtx.Lock()
			lastHash := hash[appStatus.AppID]
			nextHash, _ := hashstructure.Hash(appStatus, nil)
			hash[appStatus.AppID] = nextHash
			mtx.Unlock()
			if lastHash != nextHash {
				b, _ := json.Marshal(appStatus)
				logger.Infof("Updating app status %s", b)
			}

			app := GetHelmApp(appStatus.AppID)
			app.Status = appStatus
			app.Status.State = appstatetypes.GetState(appStatus.ResourceStates)
		})
	}

	return errors.New("app state monitor shutdown")
}
