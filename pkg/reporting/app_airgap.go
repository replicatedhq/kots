package reporting

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/store"
)

func (r *AirgapReporter) SubmitAppInfo(appID string) error {
	a, err := store.GetStore().GetApp(appID)
	if err != nil {
		if store.GetStore().IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "failed to get airgaped app")
	}

	license, err := store.GetStore().GetLatestLicenseForApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get license for airgapped app")
	}
	reportingInfo := GetReportingInfo(appID)

	err = store.GetStore().SaveReportingInfo(license.Spec.LicenseID, reportingInfo)
	if err != nil {
		return errors.Wrap(err, "failed to save reporting info")
	}

	return nil
}
