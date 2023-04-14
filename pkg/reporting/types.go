package reporting

import (
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
)

type Reporter interface {
	SubmitAppInfo(appID string) error
	SubmitPreflightData(license *kotsv1beta1.License, appID string, clusterID string, sequence int64, skipPreflights bool, installStatus storetypes.DownstreamVersionStatus, isCLI bool, preflightStatus string, appStatus string) error
}

var reporter Reporter

type AirgapReporter struct {
}

var _ Reporter = &AirgapReporter{}

type OnlineReporter struct {
}

var _ Reporter = &OnlineReporter{}
