package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/segmentio/ksuid"
)

const (
	DevEndpoint  = "http://localhost:30016"
	ProdEndpoint = "https://replicated.app"
)

type InstallMetrics struct {
	endpoint    string
	InstallID   string    `json:"install_id"`
	StartedAt   time.Time `json:"started"`
	FinishedAt  time.Time `json:"finished"`
	FailedAt    time.Time `json:"failed"`
	KotsVersion string    `json:"kots_version"`
	Cause       string    `json:"cause"`
}

func InitInstallMetrics(license *kotsv1beta1.License, disableOutboundConnections bool) *InstallMetrics {
	if disableOutboundConnections {
		return nil
	}
	m := InstallMetrics{
		endpoint:    getEndpoint(license),
		InstallID:   ksuid.New().String(),
		StartedAt:   time.Now(),
		KotsVersion: buildversion.Version(),
	}
	return &m
}

func (m *InstallMetrics) GetInstallID() string {
	if m == nil {
		return ""
	}
	return m.InstallID
}

func (m *InstallMetrics) ReportInstallStart() error {
	if m == nil || m.InstallID == "" || m.endpoint == "" {
		return nil
	}
	url := fmt.Sprintf("%s/kots_metrics/start_install/%s", m.endpoint, m.InstallID)
	return m.Post(url)
}

func (m *InstallMetrics) ReportInstallFail(cause string) error {
	if m == nil || m.InstallID == "" || m.endpoint == "" {
		return nil
	}
	m.FailedAt = time.Now()
	m.Cause = cause
	url := fmt.Sprintf("%s/kots_metrics/fail_install/%s", m.endpoint, m.InstallID)
	return m.Post(url)
}

func (m *InstallMetrics) ReportInstallFinish() error {
	if m == nil || m.InstallID == "" || m.endpoint == "" {
		return nil
	}
	m.FinishedAt = time.Now()
	url := fmt.Sprintf("%s/kots_metrics/finish_install/%s", m.endpoint, m.InstallID)
	return m.Post(url)
}

func (m *InstallMetrics) Post(url string) error {
	b, err := json.Marshal(m)
	if err != nil {
		return errors.Wrap(err, "failed to marshal json")
	}

	fmt.Println("url", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		return errors.Wrap(err, "failed to create new request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to execute post request")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func getEndpoint(license *kotsv1beta1.License) string {
	if license != nil {
		if isDevEndpoint(license.Spec.Endpoint) {
			// cluster ip services are not resolvable from the cli ...
			return DevEndpoint
		}
		return license.Spec.Endpoint
	}

	devCfgFile := fmt.Sprintf("%s/src/github.com/replicatedhq/kots/pkg/metrics/dev-cfg.json", os.Getenv("GOPATH"))
	b, err := ioutil.ReadFile(devCfgFile)
	if err == nil {
		type DevCfg struct {
			Endpoint string `json:"metrics_endpoint"`
		}
		devCfg := DevCfg{}
		json.Unmarshal([]byte(b), &devCfg)
		return devCfg.Endpoint
	}

	return ProdEndpoint
}

func isDevEndpoint(endpoint string) bool {
	result, _ := regexp.MatchString(`replicated-app`, endpoint)
	return result
}
