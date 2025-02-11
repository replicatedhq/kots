package upgradeservice

import (
	"bytes"
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
	cmd    *exec.Cmd
	port   string
	stderr bytes.Buffer
}

// map of app slug to upgrade service
var upgradeServiceMap = map[string]*UpgradeService{}
var upgradeServiceMtx = &sync.Mutex{}

// Start spins up an upgrade service for an app in the background on a random port and waits for it to be ready.
// If an upgrade service is already running for the app, it will be stopped and a new one will be started.
func Start(params types.UpgradeServiceParams) (finalError error) {
	svc, err := start(params)
	if err != nil {
		return errors.Wrap(err, "create new upgrade service")
	}
	if err := svc.waitForReady(params.AppSlug); err != nil {
		return errors.Wrap(err, "wait for upgrade service to become ready")
	}
	return nil
}

// Stop stops the upgrade service for an app.
func Stop(appSlug string) {
	upgradeServiceMtx.Lock()
	defer upgradeServiceMtx.Unlock()

	svc, _ := upgradeServiceMap[appSlug]
	if svc != nil {
		svc.stop()
	}
	delete(upgradeServiceMap, appSlug)
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
		logger.Error(errors.Wrap(err, "parse upgrade service url"))
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

	// create a new service
	newSvc := &UpgradeService{
		port:   params.Port,
		stderr: bytes.Buffer{},
	}

	paramsYAML, err := yaml.Marshal(params)
	if err != nil {
		return nil, errors.Wrap(err, "marshal params")
	}

	cmd := exec.Command(params.UpdateKOTSBin, "upgrade-service", "start", "-")
	cmd.Stdin = strings.NewReader(string(paramsYAML))
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &newSvc.stderr)
	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "start")
	}

	// calling wait helps populate the process state and reap the zombie process
	go cmd.Wait()

	// update cmd and register the service
	newSvc.cmd = cmd
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
			return errors.Errorf("upgrade service terminated. ping error: %v: stderr: %s", lasterr, s.stderr.String())
		}
		request, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%s/api/v1/upgrade-service/app/%s/ping", s.port, appSlug), nil)
		if err != nil {
			lasterr = errors.Wrap(err, "create request")
			continue
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			lasterr = errors.Wrap(err, "do request")
			continue
		}
		if response.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(response.Body)
			return errors.Errorf("unexpected status code %d: %s", response.StatusCode, string(body))
		}
		return nil
	}
}
