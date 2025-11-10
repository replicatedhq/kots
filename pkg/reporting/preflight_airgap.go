package reporting

import (
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/buildversion"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
)

func (r *AirgapReporter) SubmitPreflightData(license *licensewrapper.LicenseWrapper, appID string, clusterID string, sequence int64, skipPreflights bool, installStatus storetypes.DownstreamVersionStatus, isCLI bool, preflightStatus string, appStatus string) error {
	app, err := r.store.GetApp(appID)
	if err != nil {
		if r.store.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "failed to get airgapped app")
	}

	report := &PreflightReport{
		Events: []PreflightReportEvent{
			{
				ReportedAt:      time.Now().UTC().UnixMilli(),
				LicenseID:       license.GetLicenseID(),
				InstanceID:      appID,
				ClusterID:       clusterID,
				Sequence:        sequence,
				SkipPreflights:  skipPreflights,
				InstallStatus:   string(installStatus),
				IsCLI:           isCLI,
				PreflightStatus: preflightStatus,
				AppStatus:       appStatus,
				UserAgent:       buildversion.GetUserAgent(),
			},
		},
	}

	if err := AppendReport(r.clientset, util.PodNamespace, app.Slug, report); err != nil {
		return errors.Wrap(err, "failed to append preflight report")
	}

	return nil
}
