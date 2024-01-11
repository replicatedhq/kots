package reporting

import (
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/buildversion"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

func (r *AirgapReporter) SubmitPreflightData(license *kotsv1beta1.License, appID string, clusterID string, sequence int64, skipPreflights bool, installStatus storetypes.DownstreamVersionStatus, isCLI bool, preflightStatus string, appStatus string) error {
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
				LicenseID:       license.Spec.LicenseID,
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
