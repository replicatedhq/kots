package reporting

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
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

	headers["X-Replicated-KurlNodeCountTotal"] = strconv.Itoa(reportingInfo.KurlNodeCountTotal)
	headers["X-Replicated-KurlNodeCountReady"] = strconv.Itoa(reportingInfo.KurlNodeCountReady)

	headers["X-Replicated-IsGitOpsEnabled"] = strconv.FormatBool(reportingInfo.IsGitOpsEnabled)
	headers["X-Replicated-GitOpsProvider"] = reportingInfo.GitOpsProvider

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

func AirgapReportSecret(name string, namespace string, kotsadmUID apimachinerytypes.UID, data []byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			// since this secret is created by the kotsadm deployment, we should set the owner reference
			// so that it is deleted when the kotsadm deployment is deleted
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "kotsadm",
					UID:        kotsadmUID,
				},
			},
		},
		Data: map[string][]byte{
			InstanceReportSecretKey: data,
		},
	}
}

// EncodeAirgapReport marshals, compresses, and base64 encodes the given report
func EncodeAirgapReport(r any) ([]byte, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal airgap report")
	}
	compressedData, err := util.GzipData(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to gzip airgap report")
	}
	encodedData := base64.StdEncoding.EncodeToString(compressedData)

	return []byte(encodedData), nil
}

// DecodeAirgapReport base64 decodes, uncompresses, and unmarshals the given report
func DecodeAirgapReport(encodedData []byte, r any) error {
	decodedData, err := base64.StdEncoding.DecodeString(string(encodedData))
	if err != nil {
		return errors.Wrap(err, "failed to decode airgap report")
	}
	decompressedData, err := util.GunzipData(decodedData)
	if err != nil {
		return errors.Wrap(err, "failed to gunzip airgap report")
	}

	if err := json.Unmarshal(decompressedData, r); err != nil {
		return errors.Wrap(err, "failed to unmarshal airgap report")
	}

	return nil
}
