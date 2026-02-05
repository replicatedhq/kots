package upgradeservice

import (
	"fmt"
	"os"
	"testing"

	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
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

func TestEnsureReplicatedAppEndpointSet(t *testing.T) {
	tests := []struct {
		name            string
		existingEnv     string
		licenseEndpoint string
		expectedSet     string
	}{
		{
			name:            "sets endpoint from license",
			existingEnv:     "",
			licenseEndpoint: "https://replicated.app",
			expectedSet:     "https://replicated.app",
		},
		{
			name:            "sets default endpoint when license has no endpoint",
			existingEnv:     "",
			licenseEndpoint: "",
			expectedSet:     "https://replicated.app",
		},
		{
			name:            "does not change when REPLICATED_APP_ENDPOINT already set",
			existingEnv:     "https://existing.replicated.app",
			licenseEndpoint: "https://license.replicated.app",
			expectedSet:     "https://existing.replicated.app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup initial REPLICATED_APP_ENDPOINT state
			t.Setenv("REPLICATED_APP_ENDPOINT", tt.existingEnv)

			// Create license
			licenseData := `apiVersion: kots.io/v1beta1
kind: License
metadata:
  name: test
spec:
  appSlug: test-app
  licenseID: test-id`

			if tt.licenseEndpoint != "" {
				licenseData = fmt.Sprintf(`apiVersion: kots.io/v1beta1
kind: License
metadata:
  name: test
spec:
  endpoint: %s
  appSlug: test-app
  licenseID: test-id`, tt.licenseEndpoint)
			}

			licenseWrapper, err := licensewrapper.LoadLicenseFromBytes([]byte(licenseData))
			require.NoError(t, err)

			// Execute the function - should not panic
			ensureReplicatedAppEndpointSet(licenseWrapper)

			require.NotPanics(t, func() {
				ensureReplicatedAppEndpointSet(licenseWrapper)
			})

			// Verify env var was set correctly
			assert.Equal(t, tt.expectedSet, os.Getenv("REPLICATED_APP_ENDPOINT"))
		})
	}
}
