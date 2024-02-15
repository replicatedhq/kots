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
	EmbeddedClusterID         string `json:"embedded_cluster_id,omitempty"`
	EmbeddedClusterVersion    string `json:"embedded_cluster_version,omitempty"`
	IsGitOpsEnabled           bool   `json:"is_gitops_enabled"`
	GitOpsProvider            string `json:"gitops_provider"`
	SnapshotProvider          string `json:"snapshot_provider"`
	SnapshotFullSchedule      string `json:"snapshot_full_schedule"`
	SnapshotFullTTL           string `json:"snapshot_full_ttl"`
	SnapshotPartialSchedule   string `json:"snapshot_partial_schedule"`
	SnapshotPartialTTL        string `json:"snapshot_partial_ttl"`
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

func (r *InstanceReport) GetEventLimit() int {
	return ReportEventLimit
}

func (r *InstanceReport) GetSizeLimit() int {
	return ReportSizeLimit
}

func (r *InstanceReport) GetMtx() *sync.Mutex {
	return &instanceReportMtx
}
