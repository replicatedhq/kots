package upgradeservice

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
	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
)

var upgradeServiceProcess *os.Process
var upgradeServicePort string

// Start will spin up an upgrade service in the background on a random port.
// If an upgrade service is already running, it will be stopped and a new one will be started.
// The KOTS binary of the specified version will be used to start the upgrade service.
func Start(opts types.StartOptions) (finalError error) {
	defer func() {
		if finalError != nil {
			stop()
		}
	}()

	// stop the upgrade service if it's already running.
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
		// kotsBin, // TODO NOW: use target binary
		"/kots",
		"start-upgrade-service",
		"--port", freePort,

		"--app-id", opts.App.ID,
		"--app-slug", opts.App.Slug,
		"--app-is-airgap", fmt.Sprintf("%t", opts.App.IsAirgap),
		"--app-is-gitops", fmt.Sprintf("%t", opts.App.IsGitOps),
		"--app-license", opts.App.License, // TODO NOW: change to base64 or a file?

		"--base-archive", opts.BaseArchive,
		"--base-sequence", fmt.Sprintf("%d", opts.BaseSequence),
		"--next-sequence", fmt.Sprintf("%d", opts.NextSequence),

		"--update-cursor", opts.UpdateCursor,

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

	upgradeServicePort = freePort
	upgradeServiceProcess = cmd.Process

	if err := waitForReady(time.Second * 30); err != nil {
		return errors.Wrap(err, "failed to wait for upgrade service to become ready")
	}

	return nil
}

// Proxy will proxy the request to the upgrade service.
func Proxy(w http.ResponseWriter, r *http.Request) {
	if upgradeServicePort == "" {
		logger.Error(errors.New("upgrade service is not running"))
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	remote, err := url.Parse(fmt.Sprintf("http://localhost:%s", upgradeServicePort))
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to parse upgrade service url"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.ServeHTTP(w, r)
}

func stop() {
	if upgradeServiceProcess != nil {
		if err := upgradeServiceProcess.Signal(os.Interrupt); err != nil {
			logger.Errorf("Failed to stop upgrade service process on port %s", upgradeServicePort)
		}
	}
	upgradeServicePort = ""
	upgradeServiceProcess = nil
}

func waitForReady(timeout time.Duration) error {
	start := time.Now()

	// TODO NOW: return last error
	for {
		url := fmt.Sprintf("http://localhost:%s/api/v1/upgrade-service/ping", upgradeServicePort)
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
			return errors.Errorf("Timeout waiting for upgrade-service to become ready on port %s", upgradeServicePort)
		}
	}
}
