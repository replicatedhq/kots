package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/handlers"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/viper"
)

// Checks the KOTS CLI version against the API version
func cliVersionCheck(log *logger.CLILogger) error {
	v := viper.GetViper()

	stopCh := make(chan struct{})
	defer close(stopCh)

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
	if err != nil {
		return errors.Wrap(err, "failed to get namespace")
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

	return CompareVersions(buildversion.Version(), healthz.Version, log)
}

func CompareVersions(cliVersion string, apiVersion string, log *logger.CLILogger) error {
	cliSemver, err := semver.ParseTolerant(cliVersion)
	if err != nil {
		return errors.Wrap(err, "failed to parse cli semver")
	}
	apiSemver, err := semver.ParseTolerant(apiVersion)
	if err != nil {
		return errors.Wrap(err, "failed to parse api semver")
	}
	if cliSemver.String() != apiSemver.String() {
		updateCmd := fmt.Sprintf("curl https://kots.io/install/%s | bash", apiSemver.String())
		log.Errorf("KOTS CLI version %s does not match API version %s. To update, run:\n  $ %s\n", cliSemver.String(), apiSemver.String(), updateCmd)
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

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read")
	}

	healthz := &handlers.HealthzResponse{}
	if err := json.Unmarshal(b, healthz); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal healthz response")
	}

	return healthz, nil
}
