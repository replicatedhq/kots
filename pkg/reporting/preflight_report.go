package reporting

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

var preflightReportMtx = sync.Mutex{}

type PreflightReport struct {
	Events []PreflightReportEvent `json:"events"`
}

type PreflightReportEvent struct {
	ReportedAt      int64  `json:"reported_at"`
	LicenseID       string `json:"license_id"`
	InstanceID      string `json:"instance_id"`
	ClusterID       string `json:"cluster_id"`
	Sequence        int64  `json:"sequence"`
	SkipPreflights  bool   `json:"skip_preflights"`
	InstallStatus   string `json:"install_status"`
	IsCLI           bool   `json:"is_cli"`
	PreflightStatus string `json:"preflight_status"`
	AppStatus       string `json:"app_status"`
	UserAgent       string `json:"user_agent"`
}

func (r *PreflightReport) GetType() ReportType {
	return ReportTypePreflight
}

func (r *PreflightReport) GetSecretName(appSlug string) string {
	return fmt.Sprintf(ReportSecretNameFormat, fmt.Sprintf("%s-%s", appSlug, r.GetType()))
}

func (r *PreflightReport) GetSecretKey() string {
	return ReportSecretKey
}

func (r *PreflightReport) AppendEvents(report Report) error {
	reportToAppend, ok := report.(*PreflightReport)
	if !ok {
		return errors.Errorf("report is not a preflight report")
	}

	r.Events = append(r.Events, reportToAppend.Events...)
	if len(r.Events) > r.GetEventLimit() {
		r.Events = r.Events[len(r.Events)-r.GetEventLimit():]
	}

	// remove one event at a time until the report is under the size limit
	encoded, err := EncodeReport(r)
	if err != nil {
		return errors.Wrap(err, "failed to encode report")
	}
	for len(encoded) > r.GetSizeLimit() {
		r.Events = r.Events[1:]
		if len(r.Events) == 0 {
			return errors.Errorf("size of latest event exceeds report size limit")
		}
		encoded, err = EncodeReport(r)
		if err != nil {
			return errors.Wrap(err, "failed to encode report")
		}
	}

	return nil
}

func (r *PreflightReport) GetEventLimit() int {
	return ReportEventLimit
}

func (r *PreflightReport) GetSizeLimit() int {
	return ReportSizeLimit
}

func (r *PreflightReport) GetMtx() *sync.Mutex {
	return &preflightReportMtx
}
