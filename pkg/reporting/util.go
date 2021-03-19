package reporting

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/replicatedhq/kots/pkg/api/reporting/types"
)

func InjectReportingInfoHeaders(req *http.Request, reportingInfo *types.ReportingInfo) {
	if reportingInfo == nil {
		return
	}

	req.Header.Set("X-Replicated-K8sVersion", reportingInfo.K8sVersion)
	req.Header.Set("X-Replicated-IsKurl", fmt.Sprintf("%t", reportingInfo.IsKurl))
	req.Header.Set("X-Replicated-AppStatus", reportingInfo.AppStatus)
	req.Header.Set("X-Replicated-ClusterID", reportingInfo.ClusterID)
	req.Header.Set("X-Replicated-InstanceID", reportingInfo.InstanceID)

	if reportingInfo.Downstream.Cursor != "" {
		req.Header.Set("X-Replicated-DownstreamChannelSequence", reportingInfo.Downstream.Cursor)
	}
	if reportingInfo.Downstream.ChannelID != "" {
		req.Header.Set("X-Replicated-DownstreamChannelID", reportingInfo.Downstream.ChannelID)
	} else if reportingInfo.Downstream.ChannelName != "" {
		req.Header.Set("X-Replicated-DownstreamChannelName", reportingInfo.Downstream.ChannelName)
	}

	if kotsInstallID := os.Getenv("KOTS_INSTALL_ID"); kotsInstallID != "" {
		req.Header.Set("X-Replicated-KotsInstallID", kotsInstallID)
	}
	if kurlInstallID := os.Getenv("KURL_INSTALL_ID"); kurlInstallID != "" {
		req.Header.Set("X-Replicated-KurlInstallID", kurlInstallID)
	}
}

func canReport(endpoint string) bool {
	disableOutboundConnections := false
	// ignore the error, default to false
	disableOutboundConnections, _ = strconv.ParseBool(os.Getenv("DISABLE_OUTBOUND_CONNECTIONS"))
	if disableOutboundConnections {
		return false
	}
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
