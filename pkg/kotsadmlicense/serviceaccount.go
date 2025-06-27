package license

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type ServiceAccountToken struct {
	Identity string `json:"i"`
	Secret   string `json:"s"`
}

func ValidateServiceAccountToken(token, currentLicenseID string) (*ServiceAccountToken, error) {
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

	currentIdentity, err := extractIdentityFromLicenseID(currentLicenseID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract current license identity: %w", err)
	}

	if saToken.Identity != currentIdentity {
		return nil, fmt.Errorf("Identity mismatch: token identity does not match current license identity")
	}

	return &saToken, nil
}

func extractIdentityFromLicenseID(licenseID string) (string, error) {
	if decoded, err := base64.StdEncoding.DecodeString(licenseID); err == nil {
		var token ServiceAccountToken
		if err := json.Unmarshal(decoded, &token); err == nil {
			return token.Identity, nil
		}
	}

	return licenseID, nil
}
