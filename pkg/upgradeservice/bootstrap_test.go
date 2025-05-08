package upgradeservice

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateWithinKubeRange(t *testing.T) {
	tests := []struct {
		name           string
		params         types.UpgradeServiceParams
		expectError    bool
		expectedErrMsg string
	}{
		{
			name: "same version",
			params: types.UpgradeServiceParams{
				CurrentECVersion: "2.4.0+k8s-1.30-rc0",
				UpdateECVersion:  "2.4.1+k8s-1.30-rc1",
			},
			expectError: false,
		},
		{
			name: "one minor version update",
			params: types.UpgradeServiceParams{
				CurrentECVersion: "2.4.0+k8s-1.30-rc0",
				UpdateECVersion:  "2.5.0+k8s-1.31-rc0",
			},
			expectError: false,
		},
		{
			name: "two minor version update",
			params: types.UpgradeServiceParams{
				CurrentECVersion: "2.4.0+k8s-1.30-rc0",
				UpdateECVersion:  "2.6.0+k8s-1.32-rc0",
			},
			expectError:    true,
			expectedErrMsg: "cannot update more than one minor version",
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
			expectedErrMsg: "failed to extract current kube version",
		},
		{
			name: "invalid update version format",
			params: types.UpgradeServiceParams{
				CurrentECVersion: "2.4.0+k8s-1.30-rc0",
				UpdateECVersion:  "2.5.0-invalid-format",
			},
			expectError:    true,
			expectedErrMsg: "failed to extract update kube version",
		},
		{
			name: "downgrade version",
			params: types.UpgradeServiceParams{
				CurrentECVersion: "2.4.0+k8s-1.31-rc0",
				UpdateECVersion:  "2.5.0+k8s-1.30-rc0",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := updateWithinKubeRange(tt.params)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExtractKubeVersion(t *testing.T) {
	tests := []struct {
		name            string
		ecVersion       string
		expectedVersion string
		expectError     bool
	}{
		{
			name:            "valid format",
			ecVersion:       "2.4.0+k8s-1.30-rc0",
			expectedVersion: "1.30.0",
			expectError:     false,
		},
		{
			name:            "another valid format",
			ecVersion:       "3.1.5+k8s-1.29",
			expectedVersion: "1.29.0",
			expectError:     false,
		},
		{
			name:            "ec version with a v prefix is valid",
			ecVersion:       "v3.1.5+k8s-1.29",
			expectedVersion: "1.29.0",
			expectError:     false,
		},
		{
			name:        "invalid format - missing k8s tag",
			ecVersion:   "2.4.0-rc0",
			expectError: true,
		},
		{
			name:        "invalid format - malformed version",
			ecVersion:   "2.4.0+k8s-abc-rc0",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := extractKubeVersion(tt.ecVersion)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedVersion, version.String())
			}
		})
	}
}
