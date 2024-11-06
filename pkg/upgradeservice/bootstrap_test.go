package upgradeservice

import (
	_ "embed"
	"testing"

	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

//go:embed testassets/license.yaml
var testLicense string

func Test_bootstrap(t *testing.T) {
	tests := []struct {
		name          string
		mockClientset func() kubernetes.Interface
		mockPullFunc  func(upstreamURI string, pullOptions pull.PullOptions) (string, error)
	}{
		{
			name: "does not error when version needs config",
			mockClientset: func() kubernetes.Interface {
				return fake.NewSimpleClientset()
			},
			mockPullFunc: func(upstreamURI string, pullOptions pull.PullOptions) (string, error) {
				return "", pull.ErrConfigNeeded
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sutil.Set(&MockK8sutil{
				ClientFunc: tt.mockClientset,
			})
			pull.Set(&MockPuller{
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

type MockK8sutil struct {
	ClientFunc func() kubernetes.Interface
}

func (m *MockK8sutil) GetClientset() (kubernetes.Interface, error) {
	return m.ClientFunc(), nil
}

type MockPuller struct {
	PullFunc func(upstreamURI string, pullOptions pull.PullOptions) (string, error)
}

func (m *MockPuller) Pull(upstreamURI string, pullOptions pull.PullOptions) (string, error) {
	return m.PullFunc(upstreamURI, pullOptions)
}
