package reporting

import (
	"github.com/pkg/errors"
	reportingtypes "github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

func (r *AirgapReporter) SubmitPreflightData(license *kotsv1beta1.License, appID string, clusterID string, sequence int64, skipPreflights bool, installStatus storetypes.DownstreamVersionStatus, isCLI bool, preflightStatus string, appStatus string) error {
	status := &reportingtypes.PreflightStatus{
		InstanceID:      appID,
		ClusterID:       clusterID,
		Sequence:        sequence,
		SkipPreflights:  skipPreflights,
		InstallStatus:   string(installStatus),
		IsCLI:           isCLI,
		PreflightStatus: preflightStatus,
		AppStatus:       preflightStatus,
		KOTSVersion:     buildversion.Version(),
	}
	err := store.GetStore().SavePreflightReport(license.Spec.LicenseID, status)
	if err != nil {
		return errors.Wrap(err, "failed to save preflight report")
	}

	return nil
}
