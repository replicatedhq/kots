package reporting

import (
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/buildversion"
)

func InjectReportingInfoHeaders(req *http.Request, reportingInfo *types.ReportingInfo) {
	headers := GetReportingInfoHeaders(reportingInfo)

	for key, value := range headers {
		req.Header.Set(key, value)
	}
}

func GetReportingInfoHeaders(reportingInfo *types.ReportingInfo) map[string]string {
	headers := make(map[string]string)

	if reportingInfo == nil {
		return headers
	}

	if reportingInfo.UserAgent != "" {
		headers["User-Agent"] = reportingInfo.UserAgent
	} else {
		headers["User-Agent"] = buildversion.GetUserAgent()
	}
	headers["X-Replicated-K8sVersion"] = reportingInfo.K8sVersion
	headers["X-Replicated-IsKurl"] = strconv.FormatBool(reportingInfo.IsKurl)
	headers["X-Replicated-AppStatus"] = reportingInfo.AppStatus
	headers["X-Replicated-ClusterID"] = reportingInfo.ClusterID
	headers["X-Replicated-InstanceID"] = reportingInfo.InstanceID
	headers["X-Replicated-ReplHelmInstalls"] = strconv.Itoa(reportingInfo.Downstream.ReplHelmInstalls)
	headers["X-Replicated-NativeHelmInstalls"] = strconv.Itoa(reportingInfo.Downstream.NativeHelmInstalls)

	if reportingInfo.Downstream.Cursor != "" {
		headers["X-Replicated-DownstreamChannelSequence"] = reportingInfo.Downstream.Cursor
	}
	if reportingInfo.Downstream.ChannelID != "" {
		headers["X-Replicated-DownstreamChannelID"] = reportingInfo.Downstream.ChannelID
	} else if reportingInfo.Downstream.ChannelName != "" {
		headers["X-Replicated-DownstreamChannelName"] = reportingInfo.Downstream.ChannelName
	}

	if reportingInfo.Downstream.Status != "" {
		headers["X-Replicated-InstallStatus"] = reportingInfo.Downstream.Status
	}
	if reportingInfo.Downstream.PreflightState != "" {
		headers["X-Replicated-PreflightStatus"] = reportingInfo.Downstream.PreflightState
	}
	if reportingInfo.Downstream.Sequence != nil {
		headers["X-Replicated-DownstreamSequence"] = strconv.FormatInt(*reportingInfo.Downstream.Sequence, 10)
	}
	if reportingInfo.Downstream.Source != "" {
		headers["X-Replicated-DownstreamSource"] = reportingInfo.Downstream.Source
	}
	headers["X-Replicated-SkipPreflights"] = strconv.FormatBool(reportingInfo.Downstream.SkipPreflights)

	if reportingInfo.KOTSInstallID != "" {
		headers["X-Replicated-KotsInstallID"] = reportingInfo.KOTSInstallID
	}
	if reportingInfo.KURLInstallID != "" {
		headers["X-Replicated-KurlInstallID"] = reportingInfo.KURLInstallID
	}
	if reportingInfo.EmbeddedClusterID != "" {
		headers["X-Replicated-EmbeddedClusterID"] = reportingInfo.EmbeddedClusterID
	}
	if reportingInfo.EmbeddedClusterVersion != "" {
		headers["X-Replicated-EmbeddedClusterVersion"] = reportingInfo.EmbeddedClusterVersion
	}

	headers["X-Replicated-KurlNodeCountTotal"] = strconv.Itoa(reportingInfo.KurlNodeCountTotal)
	headers["X-Replicated-KurlNodeCountReady"] = strconv.Itoa(reportingInfo.KurlNodeCountReady)

	headers["X-Replicated-IsGitOpsEnabled"] = strconv.FormatBool(reportingInfo.IsGitOpsEnabled)
	headers["X-Replicated-GitOpsProvider"] = reportingInfo.GitOpsProvider

	headers["X-Replicated-SnapshotProvider"] = reportingInfo.SnapshotProvider
	headers["X-Replicated-SnapshotFullSchedule"] = reportingInfo.SnapshotFullSchedule
	headers["X-Replicated-SnapshotFullTTL"] = reportingInfo.SnapshotFullTTL
	headers["X-Replicated-SnapshotPartialSchedule"] = reportingInfo.SnapshotPartialSchedule
	headers["X-Replicated-SnapshotPartialTTL"] = reportingInfo.SnapshotPartialTTL

	if reportingInfo.K8sDistribution != "" {
		headers["X-Replicated-K8sDistribution"] = reportingInfo.K8sDistribution
	}

	return headers
}

func canReport(endpoint string) bool {
	if os.Getenv("KOTSADM_ENV") == "dev" && !isDevEndpoint(endpoint) {
		// don't send reports from our dev env to our production services even if this is a production license
		return false
	}
	return true
}

func isDevEndpoint(endpoint string) bool {
	result, _ := regexp.MatchString(`replicated-app`, endpoint)
	return result
}
