package license

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
)

type ServiceAccountToken struct {
	Identity string `json:"i"`
	Secret   string `json:"s"`
}

// ValidateServiceAccountToken checks if the service account token is valid and if it matches the current license identity.
// It returns the service account token, a boolean indicating if the secret has been updated, and an error if any.
func ValidateServiceAccountToken(token string, currentLicense *licensewrapper.LicenseWrapper) (*ServiceAccountToken, bool, error) {
	if currentLicense == nil || currentLicense.IsEmpty() {
		return nil, false, fmt.Errorf("current license is required")
	}

	newToken, err := extractIdentityFromToken(token)
	if err != nil {
		return nil, false, fmt.Errorf("failed to extract new token identity: %w", err)
	}

	currentToken, err := extractIdentityFromToken(currentLicense.GetLicenseID())
	if err != nil {
		return nil, false, fmt.Errorf("failed to extract current license identity: %w", err)
	}

	if newToken.Identity != currentToken.Identity {
		return nil, false, fmt.Errorf("Identity mismatch: token identity does not match current license identity")
	}

	return newToken, newToken.Secret != currentToken.Secret, nil
}

// extractIdentityFromToken extracts the identity from the provided token.
// It returns an error if the provided string is not a valid service account token.
func extractIdentityFromToken(token string) (*ServiceAccountToken, error) {
	tokenBytes, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("failed to decode service account token: %w", err)
	}

	var saToken ServiceAccountToken
	if err := json.Unmarshal(tokenBytes, &saToken); err != nil {
		return nil, fmt.Errorf("failed to parse service account token: %w", err)
	}

	if saToken.Identity == "" {
		return nil, fmt.Errorf("service account token missing identity")
	}

	if saToken.Secret == "" {
		return nil, fmt.Errorf("service account token missing secret")
	}

	return &saToken, nil
}
