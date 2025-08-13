package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestUpdateWithinKubeRange(t *testing.T) {
	tests := []struct {
		name             string
		currentECVersion string
		updateECVersion  string
		expectError      bool
		expectedErrMsg   string
	}{
		{
			name:             "same version",
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			updateECVersion:  "2.4.1+k8s-1.30-rc1",
			expectError:      false,
		},
		{
			name:             "one minor version upgrade",
			currentECVersion: "2,4.0+k8s-1.30-rc0",
			updateECVersion:  "2.5.0+k8s-1.31-rc0",
			expectError:      false,
		},
		{
			name:             "two minor version upgrade",
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			updateECVersion:  "2.6.0+k8s-1.32-rc0",
			expectError:      true,
			expectedErrMsg:   "cannot update by more than one kubernetes minor version",
		},
		{
			name:             "one minor version downgrade",
			currentECVersion: "2.4.0+k8s-1.31-rc0",
			updateECVersion:  "2.5.0+k8s-1.30-rc0",
			expectError:      true,
			expectedErrMsg:   "cannot downgrade the kubernetes version",
		},
		{
			name:             "major version mismatch",
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			updateECVersion:  "2.5.0+k8s-2.31-rc0",
			expectError:      true,
			expectedErrMsg:   "major version mismatch",
		},
		{
			name:             "invalid current version format",
			currentECVersion: "2.4.0-invalid-format",
			updateECVersion:  "2.5.0+k8s-1.31-rc0",
			expectError:      true,
			expectedErrMsg:   "failed to extract kube version",
		},
		{
			name:             "invalid update version format",
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			updateECVersion:  "2.5.0-invalid-format",
			expectError:      true,
			expectedErrMsg:   "failed to extract kube version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpdateWithinKubeRange(tt.currentECVersion, tt.updateECVersion)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
