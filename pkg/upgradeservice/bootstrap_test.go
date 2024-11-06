package upgradeservice

import (
	_ "embed"
	"testing"

	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
)

//go:embed testassets/license.yaml
var testLicense string

func Test_bootstrap(t *testing.T) {
	pull.SetPuller(&MockPuller{
		PullFunc: func(upstreamURI string, pullOptions pull.PullOptions) (string, error) {
			return "", pull.ErrConfigNeeded
		},
	})

	params := types.UpgradeServiceParams{
		AppLicense: testLicense,
		AppArchive: t.TempDir(),
	}

	err := bootstrap(params)
	if err != nil {
		t.Errorf("expected no error when ErrConfigNeeded is returned, got %v", err)
	}
}

type MockPuller struct {
	PullFunc func(upstreamURI string, pullOptions pull.PullOptions) (string, error)
}

func (m *MockPuller) Pull(upstreamURI string, pullOptions pull.PullOptions) (string, error) {
	return m.PullFunc(upstreamURI, pullOptions)
}
