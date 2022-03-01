package cli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/handlers"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/viper"
)

// Checks the KOTS CLI version against the API version
func cliVersionCheck() error {
	cliSemver, err := semver.ParseTolerant(buildversion.Version())
	if err != nil {
		return errors.Wrap(err, "failed to get cli semver")
	}

	v := viper.GetViper()

	log := logger.NewCLILogger()

	stopCh := make(chan struct{})
	defer close(stopCh)

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	namespace := v.GetString("namespace")
	if err := validateNamespace(namespace); err != nil {
		return errors.Wrap(err, "failed to validate namespace")
	}

	getPodName := func() (string, error) {
		return k8sutil.FindKotsadm(clientset, namespace)
	}

	localPort, errChan, err := k8sutil.PortForward(0, 3000, namespace, getPodName, false, stopCh, log)
	if err != nil {
		return errors.Wrap(err, "failed to start port forwarding")
	}

	go func() {
		select {
		case err := <-errChan:
			if err != nil {
				log.Error(err)
			}
		case <-stopCh:
		}
	}()

	getHealthzURL := fmt.Sprintf("http://localhost:%d/healthz", localPort)
	healthz, err := getHealthz(getHealthzURL)
	if err != nil {
		return errors.Wrap(err, "failed to get healthz")
	}

	if apiSemver, err := semver.ParseTolerant(healthz.Version); err == nil {
		if cliSemver.String() != apiSemver.String() {
			updateCmd := fmt.Sprintf("curl https://kots.io/install/%s | bash", apiSemver.String())
			fmt.Fprintf(os.Stderr, "KOTS CLI version %s does not match API version %s. To update, run:\n  $ %s\n", cliSemver.String(), apiSemver.String(), updateCmd)
		}
	}

	return nil
}

func getHealthz(url string) (*handlers.HealthzResponse, error) {
	newReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	newReq.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(newReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute request")
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read")
	}

	healthz := &handlers.HealthzResponse{}
	if err := json.Unmarshal(b, healthz); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal healthz response")
	}

	return healthz, nil
}
