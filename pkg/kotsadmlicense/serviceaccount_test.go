package license

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateServiceAccountToken(t *testing.T) {
	tests := []struct {
		name                string
		token               string
		currentLicenseID    string
		expectError         bool
		errorContains       string
		expectSecretUpdated bool
	}{
		{
			name:                "same token matches itself",
			token:               "eyJpIjoiMno2OTdlelpJR21aUFVQdHBnYUREV3pYMDNYIiwicyI6IjJ6NkRXbzFTaDZNUlkxWEdOTmkyNEduODZSYyJ9",
			currentLicenseID:    "eyJpIjoiMno2OTdlelpJR21aUFVQdHBnYUREV3pYMDNYIiwicyI6IjJ6NkRXbzFTaDZNUlkxWEdOTmkyNEduODZSYyJ9",
			expectError:         false,
			expectSecretUpdated: false,
		},
		{
			name:                "same identity different secret matches",
			token:               "eyJpIjoiMno2OTdlelpJR21aUFVQdHBnYUREV3pYMDNYIiwicyI6IjJ6NkRXbzFTaDZNUlkxWEdOTmkyNEduODZSYyJ9",
			currentLicenseID:    "eyJpIjoiMno2OTdlelpJR21aUFVQdHBnYUREV3pYMDNYIiwicyI6IjJ6NkZFVkd4YXp5UEhIU3BPYk5mTG53Y25COCJ9",
			expectError:         false,
			expectSecretUpdated: true,
		},
		{
			name:             "different identity does not match",
			token:            "eyJpIjoiMno2OTdlelpJR21aUFVQdHBnYUREV3pYMDNYIiwicyI6IjJ6NkZFVkd4YXp5UEhIU3BPYk5mTG53Y25BOCJ9",
			currentLicenseID: "eyJpIjoiMno0NW5UQUVvMEtUYVlaZnZ0elg4VjdoVWtuIiwicyI6IjJ6NDVuVFpUR1hsNTBweXpQRHhsNzBNWmVNYiJ9",
			expectError:      true,
			errorContains:    "Identity mismatch",
		},
		{
			name:             "invalid base64 token",
			token:            "invalid-base64!",
			currentLicenseID: "eyJpIjoiMno2OTdlelpJR21aUFVQdHBnYUREV3pYMDNYIiwicyI6IjJ6NkRXbzFTaDZNUlkxWEdOTmkyNEduODZSYyJ9",
			expectError:      true,
			errorContains:    "failed to decode service account token",
		},
		{
			name:             "invalid json token",
			token:            "aW52YWxpZC1qc29u", // base64 of "invalid-json"
			currentLicenseID: "eyJpIjoiMno2OTdlelpJR21aUFVQdHBnYUREV3pYMDNYIiwicyI6IjJ6NkRXbzFTaDZNUlkxWEdOTmkyNEduODZSYyJ9",
			expectError:      true,
			errorContains:    "failed to parse service account token",
		},
		{
			name:             "empty identity in token",
			token:            createTokenBase64("", "some-secret"),
			currentLicenseID: "eyJpIjoiMno2OTdlelpJR21aUFVQdHBnYUREV3pYMDNYIiwicyI6IjJ6NkRXbzFTaDZNUlkxWEdOTmkyNEduODZSYyJ9",
			expectError:      true,
			errorContains:    "service account token missing identity",
		},
		{
			name:             "empty secret in token",
			token:            createTokenBase64("some-identity", ""),
			currentLicenseID: "eyJpIjoiMno2OTdlelpJR21aUFVQdHBnYUREV3pYMDNYIiwicyI6IjJ6NkRXbzFTaDZNUlkxWEdOTmkyNEduODZSYyJ9",
			expectError:      true,
			errorContains:    "service account token missing secret",
		},
		{
			name:             "plain text license ID does not match token identity",
			token:            "eyJpIjoiMno2OTdlelpJR21aUFVQdHBnYUREV3pYMDNYIiwicyI6IjJ6NkRXbzFTaDZNUlkxWEdOTmkyNEduODZSYyJ9",
			currentLicenseID: "2z697ezZIGmZPUPtpgaDDWzX03X", // plain text identity
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, secretUpdated, err := ValidateServiceAccountToken(tt.token, tt.currentLicenseID)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.NotEmpty(t, result.Identity)
				assert.NotEmpty(t, result.Secret)
				assert.Equal(t, tt.expectSecretUpdated, secretUpdated)
			}
		})
	}
}

// Helper function to create base64 encoded tokens for testing
func createTokenBase64(identity, secret string) string {
	token := ServiceAccountToken{
		Identity: identity,
		Secret:   secret,
	}
	tokenBytes, _ := json.Marshal(token)
	return base64.StdEncoding.EncodeToString(tokenBytes)
}

func TestExtractIdentityFromLicenseID(t *testing.T) {
	tests := []struct {
		name        string
		licenseID   string
		expectedID  string
		expectError bool
	}{
		{
			name:        "valid base64 encoded license",
			licenseID:   "eyJpIjoiMno2OTdlelpJR21aUFVQdHBnYUREV3pYMDNYIiwicyI6IjJ6NkRXbzFTaDZNUlkxWEdOTmkyNEduODZSYyJ9",
			expectedID:  "2z697ezZIGmZPUPtpgaDDWzX03X",
			expectError: false,
		},
		{
			name:        "plain text license ID",
			licenseID:   "plain-text-license-id",
			expectError: true,
		},
		{
			name:        "invalid base64 license ID",
			licenseID:   "invalid-base64!",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractIdentityFromLicenseID(tt.licenseID)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedID, result.Identity)
			}
		})
	}
}
