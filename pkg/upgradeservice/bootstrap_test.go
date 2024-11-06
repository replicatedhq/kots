package upgradeservice

import (
	_ "embed"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	mock_k8sutil "github.com/replicatedhq/kots/pkg/k8sutil/mock"
	"github.com/replicatedhq/kots/pkg/pull"
	mock_pull "github.com/replicatedhq/kots/pkg/pull/mock"
	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

//go:embed testassets/license.yaml
var testLicense string

func Test_bootstrap(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, mockK8sutil *mock_k8sutil.MockK8sutil, mockPuller *mock_pull.MockPuller)
	}{
		{
			name: "does not error when version needs config",
			setup: func(t *testing.T, mockK8sutil *mock_k8sutil.MockK8sutil, mockPuller *mock_pull.MockPuller) {
				mockK8sutil.EXPECT().GetClientset().Return(fake.NewSimpleClientset(), nil)
				mockPuller.EXPECT().Pull(gomock.Any(), gomock.Any()).Return("", pull.ErrConfigNeeded)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// mock k8sutil
			k8sutil_ctrl := gomock.NewController(t)
			defer k8sutil_ctrl.Finish()
			mockK8sutil := mock_k8sutil.NewMockK8sutil(k8sutil_ctrl)
			k8sutil.Mock(mockK8sutil)

			// mock puller
			puller_ctrl := gomock.NewController(t)
			defer puller_ctrl.Finish()
			mockPuller := mock_pull.NewMockPuller(puller_ctrl)
			pull.Mock(mockPuller)

			if tt.setup != nil {
				tt.setup(t, mockK8sutil, mockPuller)
			}

			params := types.UpgradeServiceParams{
				AppLicense: testLicense,
				AppArchive: t.TempDir(),
			}

			err := bootstrap(params)
			require.NoError(t, err)
		})
	}
}
