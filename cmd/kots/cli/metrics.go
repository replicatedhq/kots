package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"time"

	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/version"
	"github.com/segmentio/ksuid"
)

// KotsInstallMetrics implements KotsMetricsInterface and collects various data
// at different stages of kots ops
type KotsInstallMetrics struct {
	install  InstallMetrics
	endpoint Endpoint
}

// InstallMetrics holds the data to be upstreamed and stored in sql db.
// Note: the data should match the SQL fields on vendorweb side
type InstallMetrics struct {
	AppSlug     string    `json:"app_slug"`
	InstallID   string    `json:"installer_id"`
	StartedAt   time.Time `json:"started"`
	FinishedAt  time.Time `json:"finished"`
	FailedAt    time.Time `json:"failed"`
	HostOS      string    `json:"os"`
	KotsVersion string    `json:"kots_version"`
	Cause       string    `json:"cause"`
}

// Endpoint holds the destination Endpoint to send the Metrics
type Endpoint struct {
	UpstreamEp string `json:"upstream_endpoint"`
}

//KotsMetricsInterface defines the interfaces to collect various metrics like Install, download etc
type KotsMetricsInterface interface {
	StartMetrics() error
	FailMetrics(cause string) error
	FinishMetrics() error
}

// CollectStartInstallMetrics tracks various stages during kots installation
func CollectStartInstallMetrics(im KotsMetricsInterface) {
	// During airgap, im will be nil
	if im != nil {
		im.StartMetrics()
	}
}

// CollectFailInstallMetrics tracks the kots installation failures
func CollectFailInstallMetrics(im KotsMetricsInterface, cause string) {
	// During airgap, im will be nil
	if im != nil {
		im.FailMetrics(cause)
	}
}

// CollectFinishInstallMetrics tracks successful completion of kots installation
func CollectFinishInstallMetrics(im KotsMetricsInterface) {
	// During airgap, im will be nil
	if im != nil {
		im.FinishMetrics()
	}
}

// StartMetrics implements KotsMetricsInterface
func (m *KotsInstallMetrics) StartMetrics() error {

	im := &m.install
	upstream := m.endpoint.UpstreamEp
	url := fmt.Sprintf("%s/kots_metrics/start_install/%s", upstream, im.InstallID)
	return PostMetrics(url, im)
}

// FailMetrics implements KotsMetricsInterface
func (m *KotsInstallMetrics) FailMetrics(cause string) error {
	im := &m.install
	upstream := m.endpoint.UpstreamEp
	im.FailedAt = time.Now()
	im.Cause = cause
	url := fmt.Sprintf("%s/kots_metrics/fail_install/%s", upstream, im.InstallID)
	return PostMetrics(url, im)
}

// FinishMetrics implements KotsMetricsInterface
func (m *KotsInstallMetrics) FinishMetrics() error {
	im := &m.install
	upstream := m.endpoint.UpstreamEp
	im.FinishedAt = time.Now()
	url := fmt.Sprintf("%s/kots_metrics/finish_install/%s", upstream, im.InstallID)
	return PostMetrics(url, im)
}

//PostMetrics does HTTP to post metrics
func PostMetrics(url string, im *InstallMetrics) error {
	// Marshall im and POST metrics to upstream
	jsonStr, _ := json.Marshal(*im)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	// 1sec timeout to reach the server, dont wait for too long
	client := &http.Client{
		Timeout: time.Second * 1,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// InitMetrics initializes with static data
func InitMetrics(deployOptions *kotsadmtypes.DeployOptions, isAirgap bool) (*KotsInstallMetrics, error) {

	// Dont do metrics in airgap as there is no outbound n/w
	if isAirgap == true {
		return nil, nil
	}
	m := &KotsInstallMetrics{}
	im := &m.install
	if deployOptions.License != nil {
		im.AppSlug = deployOptions.License.Spec.AppSlug
	} else {
		// if the license file is not passed, use the namespace
		im.AppSlug = deployOptions.Namespace
	}
	m.endpoint.UpstreamEp = getEndpoint(deployOptions)
	im.StartedAt = time.Now()
	// use guid as InstallID
	im.InstallID = ksuid.New().String()

	// Get the Host OS
	dist, distVersion, _ := discover()
	im.HostOS = dist + " " + distVersion
	im.KotsVersion = version.Version()
	return m, nil
}

//ProductionEp present in license file: https://replicated.app
//codeServerSvc/Dev env present in license file: file https://replicated-app
//codeServerEP use this if no license file is passed http://localhost:30016
func getEndpoint(deployOptions *kotsadmtypes.DeployOptions) string {

	cliConfigPath := "src/github.com/replicatedhq/kots/cmd/kots/cli/config"
	// if the license file is passed as argument, and if the endpoint is not a dev endpoint
	if deployOptions.License != nil {
		if isDevEndpoint(deployOptions.License.Spec.Endpoint) == false {
			return deployOptions.License.Spec.Endpoint
		}
	}
	// if license file not passed, pick the dev endpoint from the dev-config file
	devCfgFile := fmt.Sprintf("%s/%s/dev-cfg.json", os.Getenv("GOPATH"), cliConfigPath)
	file, err := ioutil.ReadFile(devCfgFile)
	if err == nil {
		m := KotsInstallMetrics{}
		ep := &m.endpoint
		_ = json.Unmarshal([]byte(file), &ep)
		return ep.UpstreamEp
	}
	// if dev-config file is not present, fallback to productionEp
	productionEp := "https://replicated.app/"
	return productionEp
}

func isDevEndpoint(endpoint string) bool {

	result, _ := regexp.MatchString(`replicated-app`, endpoint)
	return result

}
