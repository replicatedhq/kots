package upgradeservice

import (
	_ "embed"
	"testing"

	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
	"github.com/stretchr/testify/require"
)

//go:embed testassets/license.yaml
var testLicense string

func Test_bootstrap(t *testing.T) {
	tests := []struct {
		name         string
		mockPullFunc func(upstreamURI string, pullOptions pull.PullOptions) (string, error)
	}{
		{
			name: "does not error when version needs config",
			mockPullFunc: func(upstreamURI string, pullOptions pull.PullOptions) (string, error) {
				return "", pull.ErrConfigNeeded
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pull.SetPuller(&MockPuller{
				PullFunc: tt.mockPullFunc,
			})

			params := types.UpgradeServiceParams{
				AppLicense: testLicense,
				AppArchive: t.TempDir(),
			}

			err := bootstrap(params)
			require.NoError(t, err)
		})
	}
}

type MockPuller struct {
	PullFunc func(upstreamURI string, pullOptions pull.PullOptions) (string, error)
}

func (m *MockPuller) Pull(upstreamURI string, pullOptions pull.PullOptions) (string, error) {
	return m.PullFunc(upstreamURI, pullOptions)
}
