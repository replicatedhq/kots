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
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upgrader/types"
)

var upgraderProcess *os.Process
var upgraderPort string

// Start will spin up an upgrader service in the background on a random port.
// If an upgrader is already running, it will be stopped and a new one will be started.
// The KOTS binary of the specified version will be used to start the upgrader.
func Start(opts types.StartOptions) (finalError error) {
	defer func() {
		if finalError != nil {
			stop()
		}
	}()

	// stop the upgrader if it's already running.
	// don't bail if not able to stop, and start a new one
	stop()

	fp, err := freeport.GetFreePort()
	if err != nil {
		return errors.Wrap(err, "failed to get free port")
	}
	freePort := fmt.Sprintf("%d", fp)

	// TODO NOW: uncomment this
	// kotsBin, err := kotsutil.DownloadKOTSBinary(request.KOTSVersion)
	// if err != nil {
	// 	return errors.Wrapf(err, "failed to download kots binary version %s", kotsVersion)
	// }

	cmd := exec.Command(
		// kotsBin,
		"/kots",
		"start-upgrader",
		"--port", freePort,

		"--app-id", opts.App.ID,
		"--app-slug", opts.App.Slug,
		"--app-sequence", fmt.Sprintf("%d", opts.App.CurrentSequence),
		"--app-is-airgap", fmt.Sprintf("%t", opts.App.IsAirgap),
		"--app-license", opts.App.License,
		"--app-archive", opts.AppArchive,

		"--registry-endpoint", opts.RegistrySettings.Hostname,
		"--registry-username", opts.RegistrySettings.Username,
		"--registry-password", opts.RegistrySettings.Password,
		"--registry-namespace", opts.RegistrySettings.Namespace,
		"--registry-is-readonly", fmt.Sprintf("%t", opts.RegistrySettings.IsReadOnly),
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

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

// Proxy will proxy the request to the upgrader service.
func Proxy(w http.ResponseWriter, r *http.Request) {
	if upgraderPort == "" {
		logger.Error(errors.New("upgrader is not running"))
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
		url := fmt.Sprintf("http://localhost:%s/api/v1/upgrader/ping", upgraderPort)
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
