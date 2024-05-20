package upgrader

import (
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/phayes/freeport"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
)

type Upgrader struct {
	process *os.Process
	port    string
}

// Start will spin up an upgrader service in the background on a random port.
// Caller is responsible for stopping the upgrader.
// The KOTS binary of the specified version will be downloaded and used to start the upgrader.
func (u *Upgrader) Start(kotsVersion string) (finalError error) {
	if u.port != "" {
		return errors.Errorf("upgrader is already running on port %s", u.port)
	}

	defer func() {
		if finalError != nil {
			u.Stop()
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

	cmd := exec.Command(kotsBin, "upgrader", "--port", freePort)
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start")
	}

	u.port = freePort
	u.process = cmd.Process

	if err := u.WaitForReady(time.Second * 30); err != nil {
		return errors.Wrap(err, "failed to wait for upgrader to become ready")
	}

	return nil
}

func (r *Upgrader) Stop() {
	if r.process != nil {
		if err := r.process.Signal(os.Interrupt); err != nil {
			logger.Debugf("Failed to stop upgrader process on port %s", r.port)
		}
	}
	r.port = ""
	r.process = nil
}

func (r *Upgrader) WaitForReady(timeout time.Duration) error {
	start := time.Now()

	for {
		url := fmt.Sprintf("http://localhost:%s", r.port)
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
			return errors.Errorf("Timeout waiting for upgrader to become ready on port %s", r.port)
		}
	}
}

// This is only used for integration tests
func (r *Upgrader) OverridePort(port string) {
	r.port = port
}
