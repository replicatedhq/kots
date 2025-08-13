package upgradeservice

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrapUpdateWithinKubeRange(t *testing.T) {
	tests := []struct {
		name           string
		params         types.UpgradeServiceParams
		expectError    bool
		expectedErrMsg string
	}{
		{
			name: "two minor version upgrade",
			params: types.UpgradeServiceParams{
				CurrentECVersion: "2.4.0+k8s-1.30-rc0",
				UpdateECVersion:  "2.6.0+k8s-1.32-rc0",
			},
			expectError:    true,
			expectedErrMsg: "cannot update by more than one kubernetes minor version",
		},
		{
			name: "one minor version downgrade",
			params: types.UpgradeServiceParams{
				CurrentECVersion: "2.4.0+k8s-1.31-rc0",
				UpdateECVersion:  "2.5.0+k8s-1.30-rc0",
			},
			expectError:    true,
			expectedErrMsg: "cannot downgrade the kubernetes version",
		},
		{
			name: "major version mismatch",
			params: types.UpgradeServiceParams{
				CurrentECVersion: "2.4.0+k8s-1.30-rc0",
				UpdateECVersion:  "2.5.0+k8s-2.31-rc0",
			},
			expectError:    true,
			expectedErrMsg: "major version mismatch",
		},
		{
			name: "invalid current version format",
			params: types.UpgradeServiceParams{
				CurrentECVersion: "2.4.0-invalid-format",
				UpdateECVersion:  "2.5.0+k8s-1.31-rc0",
			},
			expectError:    true,
			expectedErrMsg: "failed to extract kube version",
		},
		{
			name: "invalid update version format",
			params: types.UpgradeServiceParams{
				CurrentECVersion: "2.4.0+k8s-1.30-rc0",
				UpdateECVersion:  "2.5.0-invalid-format",
			},
			expectError:    true,
			expectedErrMsg: "failed to extract kube version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bootstrap(tt.params)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
