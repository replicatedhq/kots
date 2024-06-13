package upgradeservice

import (
	_ "embed"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/phayes/freeport"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
	"gopkg.in/yaml.v3"
)

type UpgradeService struct {
	process      *os.Process
	processState *os.ProcessState
	port         string
}

// map of app slug to upgrade service
var upgradeServiceMap = map[string]UpgradeService{}
var upgradeServiceMtx = &sync.Mutex{}

// Start spins up an upgrade service for an app in the background on a random port.
// If an upgrade service is already running for the app, it will be stopped and a new one will be started.
func Start(params types.UpgradeServiceParams) (finalError error) {
	defer func() {
		if finalError != nil {
			stop(params.AppSlug)
		}
	}()

	// stop the upgrade service if it's already running.
	// don't bail if not able to stop, and start a new one
	stop(params.AppSlug)

	fp, err := freeport.GetFreePort()
	if err != nil {
		return errors.Wrap(err, "failed to get free port")
	}
	params.Port = fmt.Sprintf("%d", fp)

	paramsYAML, err := yaml.Marshal(params)
	if err != nil {
		return errors.Wrap(err, "failed to marshal params")
	}
	paramsFile, err := os.CreateTemp("", "upgrade-service-params-*.yaml")
	if err != nil {
		return errors.Wrap(err, "failed to create temp file")
	}
	defer os.Remove(paramsFile.Name())

	if _, err := paramsFile.Write(paramsYAML); err != nil {
		return errors.Wrap(err, "failed to write params to file")
	}

	// TODO NOW: use local /kots bin if:
	// - version is the same as the one running
	// - OR it's a dev env

	// TODO NOW: uncomment this
	// kotsBin, err := kotsutil.DownloadKOTSBinary(request.KOTSVersion)
	// if err != nil {
	// 	return errors.Wrapf(err, "failed to download kots binary version %s", kotsVersion)
	// }

	kotsBin := kotsutil.GetKOTSBinPath()

	cmd := exec.Command(
		// TODO NOW: use target binary
		kotsBin,
		"upgrade-service",
		"start",
		paramsFile.Name(),
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start")
	}

	addUpgradeService(params.AppSlug, UpgradeService{
		process:      cmd.Process,
		processState: cmd.ProcessState,
		port:         params.Port,
	})

	// TODO NOW: what's a good timeout here, specially for airgap?
	// bootsrapping can take a while to pull and render the archive
	if err := waitForReady(params.AppSlug, time.Minute*2); err != nil {
		return errors.Wrap(err, "failed to wait for upgrade service to become ready")
	}

	return nil
}

// Proxy proxies the request to the app's upgrade service.
func Proxy(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	if appSlug == "" {
		logger.Error(errors.New("upgrade service requires app slug in path"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	upgradeService, ok := getUpgradeService(appSlug)
	if !ok {
		logger.Error(errors.Errorf("upgrade service is not running for app %s", appSlug))
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	remote, err := url.Parse(fmt.Sprintf("http://localhost:%s", upgradeService.port))
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse upgrade service url"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.ServeHTTP(w, r)
}

// stop stops the upgrade service for the given app.
func stop(appSlug string) {
	upgradeService, ok := getUpgradeService(appSlug)
	if !ok {
		return
	}
	if upgradeService.process != nil {
		if err := upgradeService.process.Signal(os.Interrupt); err != nil {
			logger.Errorf("failed to stop upgrade service process for %s on port %s", appSlug, upgradeService.port)
		}
	}
	removeUpgradeService(appSlug)
}

func waitForReady(appSlug string, timeout time.Duration) error {
	start := time.Now()
	var lasterr error
	for {
		upgradeService, ok := getUpgradeService(appSlug)
		if !ok {
			return errors.Errorf("upgrade service was stopped. last error: %v", lasterr)
		}
		if upgradeService.processState != nil && upgradeService.processState.Exited() {
			return errors.Errorf("upgrade service process exited. last error: %v", lasterr)
		}
		if time.Sleep(time.Second); time.Since(start) > timeout {
			return errors.Errorf("Timeout waiting for upgrade service to become ready on port %s. last error: %v", upgradeService.port, lasterr)
		}
		request, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%s/api/v1/upgrade-service/app/%s/ping", upgradeService.port, appSlug), nil)
		if err != nil {
			lasterr = errors.Wrap(err, "failed to create request")
			continue
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			lasterr = errors.Wrap(err, "failed to do request")
			continue
		}
		if response.StatusCode != http.StatusOK {
			lasterr = errors.Errorf("unexpected status code %d", response.StatusCode)
			continue
		}
		return nil
	}
}

func addUpgradeService(appSlug string, upgradeService UpgradeService) {
	upgradeServiceMtx.Lock()
	upgradeServiceMap[appSlug] = upgradeService
	upgradeServiceMtx.Unlock()
}

func removeUpgradeService(appSlug string) {
	upgradeServiceMtx.Lock()
	delete(upgradeServiceMap, appSlug)
	upgradeServiceMtx.Unlock()
}

func getUpgradeService(appSlug string) (UpgradeService, bool) {
	upgradeServiceMtx.Lock()
	upgradeService, ok := upgradeServiceMap[appSlug]
	upgradeServiceMtx.Unlock()
	return upgradeService, ok
}
