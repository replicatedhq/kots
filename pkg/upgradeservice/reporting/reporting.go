package reporting

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	reportingpkg "github.com/replicatedhq/kots/pkg/reporting"
	upgradeservicetypes "github.com/replicatedhq/kots/pkg/upgradeservice/types"
	"github.com/replicatedhq/kots/pkg/util"
)

func SubmitAppInfo(params upgradeservicetypes.UpgradeServiceParams) error {
	license, err := kotsutil.LoadLicenseFromBytes([]byte(params.AppLicense))
	if err != nil {
		return errors.Wrap(err, "failed to load license from bytes")
	}

	if params.AppIsAirgap {
		clientset, err := k8sutil.GetClientset()
		if err != nil {
			return errors.Wrap(err, "failed to get clientset")
		}
		report := reportingpkg.BuildInstanceReport(license.Spec.LicenseID, params.ReportingInfo)
		return reportingpkg.AppendReport(clientset, util.PodNamespace, params.AppSlug, report)
	}
	return reportingpkg.SendOnlineAppInfo(license, params.ReportingInfo)
}
