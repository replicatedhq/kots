package upgradeservice

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
	"gopkg.in/yaml.v3"
)

type UpgradeService struct {
	cmd  *exec.Cmd
	port string
}

// map of app slug to upgrade service
var upgradeServiceMap = map[string]*UpgradeService{}
var upgradeServiceMtx = &sync.Mutex{}

// Start spins up an upgrade service for an app in the background on a random port and waits for it to be ready.
// If an upgrade service is already running for the app, it will be stopped and a new one will be started.
func Start(params types.UpgradeServiceParams) (finalError error) {
	svc, err := start(params)
	if err != nil {
		return errors.Wrap(err, "failed to create new upgrade service")
	}
	if err := svc.waitForReady(params.AppSlug); err != nil {
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

	svc, ok := upgradeServiceMap[appSlug]
	if !ok {
		logger.Error(errors.Errorf("upgrade service not found for app %s", appSlug))
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	if !svc.isRunning() {
		logger.Error(errors.Errorf("upgrade service is not running for app %s", appSlug))
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	remote, err := url.Parse(fmt.Sprintf("http://localhost:%s", svc.port))
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse upgrade service url"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.ServeHTTP(w, r)
}

func start(params types.UpgradeServiceParams) (*UpgradeService, error) {
	upgradeServiceMtx.Lock()
	defer upgradeServiceMtx.Unlock()

	// stop the current service
	currSvc, _ := upgradeServiceMap[params.AppSlug]
	if currSvc != nil {
		currSvc.stop()
	}

	paramsYAML, err := yaml.Marshal(params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal params")
	}

	cmd := exec.Command(params.UpdateKOTSBin, "upgrade-service", "start", "-")
	cmd.Stdin = strings.NewReader(string(paramsYAML))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "failed to start")
	}

	// calling wait helps populate the process state and reap the zombie process
	go cmd.Wait()

	// create a new service
	newSvc := &UpgradeService{
		cmd:  cmd,
		port: params.Port,
	}
	upgradeServiceMap[params.AppSlug] = newSvc

	return newSvc, nil
}

func (s *UpgradeService) stop() {
	if !s.isRunning() {
		return
	}
	logger.Infof("Stopping upgrade service on port %s", s.port)
	if err := s.cmd.Process.Signal(os.Interrupt); err != nil {
		logger.Errorf("Failed to stop upgrade service on port %s: %v", s.port, err)
	}
}

func (s *UpgradeService) isRunning() bool {
	return s != nil && s.cmd != nil && s.cmd.ProcessState == nil
}

func (s *UpgradeService) waitForReady(appSlug string) error {
	var lasterr error
	for {
		time.Sleep(time.Second)
		if s == nil || s.cmd == nil {
			return errors.New("upgrade service not found")
		}
		if s.cmd.ProcessState != nil {
			return errors.Errorf("upgrade service terminated. last error: %v", lasterr)
		}
		request, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%s/api/v1/upgrade-service/app/%s/ping", s.port, appSlug), nil)
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
			body, _ := io.ReadAll(response.Body)
			return errors.Errorf("unexpected status code %d: %s", response.StatusCode, string(body))
		}
		return nil
	}
}
