package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/segmentio/ksuid"
)

const (
	DevEndpoint = "http://localhost:30016"
)

type InstallMetrics struct {
	endpoint                   string
	disableOutboundConnections bool
	InstallID                  string    `json:"install_id"`
	StartedAt                  time.Time `json:"started"`
	FinishedAt                 time.Time `json:"finished"`
	FailedAt                   time.Time `json:"failed"`
	KotsVersion                string    `json:"kots_version"`
	Cause                      string    `json:"cause"`
}

func InitInstallMetrics(license *kotsv1beta1.License, disableOutboundConnections bool) (InstallMetrics, error) {
	endpoint, err := getEndpoint(license)
	if err != nil {
		return InstallMetrics{}, errors.Wrap(err, "failed to get endpoint")
	}

	m := InstallMetrics{
		endpoint:                   endpoint,
		disableOutboundConnections: disableOutboundConnections,
		InstallID:                  ksuid.New().String(),
		StartedAt:                  time.Now(),
		KotsVersion:                buildversion.Version(),
	}
	return m, nil
}

func (m InstallMetrics) ReportInstallStart() error {
	if m.endpoint == "" || m.InstallID == "" {
		return nil
	}
	url := fmt.Sprintf("%s/kots_metrics/start_install/%s", m.endpoint, m.InstallID)
	return m.Post(url)
}

func (m InstallMetrics) ReportInstallFail(cause string) error {
	if m.endpoint == "" || m.InstallID == "" {
		return nil
	}
	m.FailedAt = time.Now()
	m.Cause = cause
	url := fmt.Sprintf("%s/kots_metrics/fail_install/%s", m.endpoint, m.InstallID)
	return m.Post(url)
}

func (m InstallMetrics) ReportInstallFinish() error {
	if m.endpoint == "" || m.InstallID == "" {
		return nil
	}
	m.FinishedAt = time.Now()
	url := fmt.Sprintf("%s/kots_metrics/finish_install/%s", m.endpoint, m.InstallID)
	return m.Post(url)
}

func (m InstallMetrics) Post(url string) error {
	if m.disableOutboundConnections {
		return nil
	}

	b, err := json.Marshal(m)
	if err != nil {
		return errors.Wrap(err, "failed to marshal json")
	}

	req, err := util.NewRequest("POST", url, bytes.NewBuffer(b))
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

func getEndpoint(license *kotsv1beta1.License) (string, error) {
	endpoint := util.ReplicatedAppEndpoint(license)

	if isDevEndpoint(endpoint) {
		// cluster ip services are not resolvable from the cli ...
		return DevEndpoint, nil
	}

	return endpoint, nil
}

func isDevEndpoint(endpoint string) bool {
	result, _ := regexp.MatchString(`replicated-app`, endpoint)
	return result
}
