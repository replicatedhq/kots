package ocistore

import (
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	downstreamtypes "github.com/replicatedhq/kots/kotsadm/pkg/downstream/types"
)

func (s OCIStore) AddAppToAllDownstreams(appID string) error {
	return errors.New("not implemented")
}

func (s OCIStore) ListInstalledApps() ([]*apptypes.App, error) {
	return nil, errors.New("not implemented")
}

func (s OCIStore) GetApp(id string) (*apptypes.App, error) {
	return nil, errors.New("not implemented")
}

func (s OCIStore) GetAppFromSlug(slug string) (*apptypes.App, error) {
	return nil, errors.New("not implemented")
}

func (s OCIStore) CreateApp(name string, upstreamURI string, licenseData string, isAirgapEnabled bool) (*apptypes.App, error) {
	return nil, errors.New("not implemented")
}

func (c OCIStore) ListDownstreamsForApp(appID string) ([]downstreamtypes.Downstream, error) {
	return nil, errors.New("not implemented")
}

func (c OCIStore) GetDownstream(clusterID string) (*downstreamtypes.Downstream, error) {
	return nil, errors.New("not implemented")
}

func (c OCIStore) IsGitOpsEnabledForApp(appID string) (bool, error) {
	return false, errors.New("not implemented")
}

func (c OCIStore) SetUpdateCheckerSpec(appID string, updateCheckerSpec string) error {
	return errors.New("not implemented")
}

func (c OCIStore) SetPreflightResults(appID string, sequence int64, results []byte) error {
	return errors.New("not implemented")
}
