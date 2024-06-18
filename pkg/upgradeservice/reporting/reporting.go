package reporting

import (
	"github.com/pkg/errors"
	reportingtypes "github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	reportingpkg "github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

func SubmitAppInfo(reportingInfo *reportingtypes.ReportingInfo, appSlug string, license *kotsv1beta1.License, isAirgap bool) error {
	if isAirgap {
		clientset, err := k8sutil.GetClientset()
		if err != nil {
			return errors.Wrap(err, "failed to get clientset")
		}
		report := reportingpkg.BuildInstanceReport(license.Spec.LicenseID, reportingInfo)
		return reportingpkg.AppendReport(clientset, util.PodNamespace, appSlug, report)
	}
	return reportingpkg.SendOnlineAppInfo(license, reportingInfo)
}
