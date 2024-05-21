package upgrader

import (
	_ "embed"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/phayes/freeport"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
)

var upgraderProcess *os.Process
var upgraderPort string

// Init will spin up an upgrader service in the background on a random port.
// If an upgrader is already running, it will be stopped and a new one will be started.
// The KOTS binary of the specified version will be used to start the upgrader.
func Init(w http.ResponseWriter, r *http.Request) {
	// TODO NOW: get these from the request
	kotsVersion := "v1.109.3"

	// stop the upgrader if it's already running.
	// don't bail if not able to stop, and start a new one
	stop()

	if err := start(kotsVersion); err != nil {
		logger.Error(errors.Wrap(err, "failed to start upgrader"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Proxy will proxy the request to the upgrader service.
func Proxy(w http.ResponseWriter, r *http.Request) {
	if upgraderPort == "" {
		logger.Error(errors.New("upgrader port is not set"))
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	remote, err := url.Parse(fmt.Sprintf("http://localhost:%s", upgraderPort))
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse upgrader url"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.ServeHTTP(w, r)
}

func start(kotsVersion string) (finalError error) {
	if upgraderPort != "" {
		return errors.Errorf("upgrader is already running on port %s", upgraderPort)
	}

	defer func() {
		if finalError != nil {
			stop()
		}
	}()

	fp, err := freeport.GetFreePort()
	if err != nil {
		return errors.Wrap(err, "failed to get free port")
	}
	freePort := fmt.Sprintf("%d", fp)

	kotsBin, err := kotsutil.DownloadKOTSBinary(kotsVersion)
	if err != nil {
		return errors.Wrapf(err, "failed to download kots binary version %s", kotsVersion)
	}

	cmd := exec.Command(
		kotsBin,
		"admin-console",
		"upgrader",
		"--port",
		freePort,
	)

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start")
	}

	upgraderPort = freePort
	upgraderProcess = cmd.Process

	if err := waitForReady(time.Second * 30); err != nil {
		return errors.Wrap(err, "failed to wait for upgrader to become ready")
	}

	return nil
}

func stop() {
	if upgraderProcess != nil {
		if err := upgraderProcess.Signal(os.Interrupt); err != nil {
			logger.Errorf("Failed to stop upgrader process on port %s", upgraderPort)
		}
	}
	upgraderPort = ""
	upgraderProcess = nil
}

func waitForReady(timeout time.Duration) error {
	start := time.Now()

	for {
		url := fmt.Sprintf("http://localhost:%s", upgraderPort)
		newRequest, err := http.NewRequest("GET", url, nil)
		if err == nil {
			resp, err := http.DefaultClient.Do(newRequest)
			if err == nil {
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}

		time.Sleep(time.Second)

		if time.Since(start) > timeout {
			return errors.Errorf("Timeout waiting for upgrader to become ready on port %s", upgraderPort)
		}
	}
}
