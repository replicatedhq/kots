package reporting

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

var instanceReportMtx = sync.Mutex{}

type InstanceReport struct {
	Events []InstanceReportEvent `json:"events"`
}

type InstanceReportEvent struct {
	ReportedAt                int64  `json:"reported_at"`
	LicenseID                 string `json:"license_id"`
	InstanceID                string `json:"instance_id"`
	ClusterID                 string `json:"cluster_id"`
	AppStatus                 string `json:"app_status"`
	IsKurl                    bool   `json:"is_kurl"`
	KurlNodeCountTotal        int    `json:"kurl_node_count_total"`
	KurlNodeCountReady        int    `json:"kurl_node_count_ready"`
	K8sVersion                string `json:"k8s_version"`
	K8sDistribution           string `json:"k8s_distribution,omitempty"`
	UserAgent                 string `json:"user_agent"`
	KotsInstallID             string `json:"kots_install_id,omitempty"`
	KurlInstallID             string `json:"kurl_install_id,omitempty"`
	IsGitOpsEnabled           bool   `json:"is_gitops_enabled"`
	GitOpsProvider            string `json:"gitops_provider"`
	DownstreamChannelID       string `json:"downstream_channel_id,omitempty"`
	DownstreamChannelSequence uint64 `json:"downstream_channel_sequence,omitempty"`
	DownstreamChannelName     string `json:"downstream_channel_name,omitempty"`
	DownstreamSequence        *int64 `json:"downstream_sequence,omitempty"`
	DownstreamSource          string `json:"downstream_source,omitempty"`
	InstallStatus             string `json:"install_status,omitempty"`
	PreflightState            string `json:"preflight_state,omitempty"`
	SkipPreflights            bool   `json:"skip_preflights"`
	ReplHelmInstalls          int    `json:"repl_helm_installs"`
	NativeHelmInstalls        int    `json:"native_helm_installs"`
}

func (r *InstanceReport) GetType() ReportType {
	return ReportTypeInstance
}

func (r *InstanceReport) GetSecretName(appSlug string) string {
	return fmt.Sprintf(ReportSecretNameFormat, fmt.Sprintf("%s-%s", appSlug, r.GetType()))
}

func (r *InstanceReport) GetSecretKey() string {
	return ReportSecretKey
}

func (r *InstanceReport) AppendEvents(report Report) error {
	reportToAppend, ok := report.(*InstanceReport)
	if !ok {
		return errors.Errorf("report is not an instance report")
	}

	r.Events = append(r.Events, reportToAppend.Events...)
	if len(r.Events) > r.GetEventLimit() {
		r.Events = r.Events[len(r.Events)-r.GetEventLimit():]
	}

	return nil
}

func (r *InstanceReport) GetEventLimit() int {
	return ReportEventLimit
}

func (r *InstanceReport) GetMtx() *sync.Mutex {
	return &instanceReportMtx
}
