package reporting

import (
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
)

var airgapAppInfoMtx sync.Mutex

func (r *AirgapReporter) SubmitAppInfo(appID string) error {
	// make sure events are reported in order
	airgapAppInfoMtx.Lock()
	defer func() {
		time.Sleep(1 * time.Second)
		airgapAppInfoMtx.Unlock()
	}()

	a, err := r.store.GetApp(appID)
	if err != nil {
		if r.store.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "failed to get airgapped app")
	}

	license, err := r.store.GetLatestLicenseForApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get license for airgapped app")
	}
	reportingInfo := GetReportingInfo(appID)

	report := BuildInstanceReport(license.Spec.LicenseID, reportingInfo)

	if err := AppendReport(r.clientset, util.PodNamespace, a.Slug, report); err != nil {
		return errors.Wrap(err, "failed to append instance report")
	}

	return nil
}

func BuildInstanceReport(licenseID string, reportingInfo *types.ReportingInfo) *InstanceReport {
	// not using the "cursor" packages because it doesn't provide access to the underlying int64
	downstreamSequence, err := strconv.ParseUint(reportingInfo.Downstream.Cursor, 10, 64)
	if err != nil {
		logger.Debugf("failed to parse downstream cursor %q: %v", reportingInfo.Downstream.Cursor, err)
	}

	return &InstanceReport{
		Events: []InstanceReportEvent{
			{
				ReportedAt:                time.Now().UTC().UnixMilli(),
				LicenseID:                 licenseID,
				InstanceID:                reportingInfo.InstanceID,
				ClusterID:                 reportingInfo.ClusterID,
				AppStatus:                 reportingInfo.AppStatus,
				IsKurl:                    reportingInfo.IsKurl,
				KurlNodeCountTotal:        reportingInfo.KurlNodeCountTotal,
				KurlNodeCountReady:        reportingInfo.KurlNodeCountReady,
				K8sVersion:                reportingInfo.K8sVersion,
				K8sDistribution:           reportingInfo.K8sDistribution,
				UserAgent:                 reportingInfo.UserAgent,
				KotsInstallID:             reportingInfo.KOTSInstallID,
				KurlInstallID:             reportingInfo.KURLInstallID,
				IsGitOpsEnabled:           reportingInfo.IsGitOpsEnabled,
				GitOpsProvider:            reportingInfo.GitOpsProvider,
				DownstreamChannelID:       reportingInfo.Downstream.ChannelID,
				DownstreamChannelSequence: downstreamSequence,
				DownstreamChannelName:     reportingInfo.Downstream.ChannelName,
				DownstreamSequence:        reportingInfo.Downstream.Sequence,
				DownstreamSource:          reportingInfo.Downstream.Source,
				InstallStatus:             reportingInfo.Downstream.Status,
				PreflightState:            reportingInfo.Downstream.PreflightState,
				SkipPreflights:            reportingInfo.Downstream.SkipPreflights,
				ReplHelmInstalls:          reportingInfo.Downstream.ReplHelmInstalls,
				NativeHelmInstalls:        reportingInfo.Downstream.NativeHelmInstalls,
			},
		},
	}
}
