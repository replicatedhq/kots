package util

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

type ServiceAccountToken struct {
	Identity string `json:"i"`
	Secret   string `json:"s"`
}

func SetLicenseAuth(req *http.Request, license *kotsv1beta1.License) {
	username, password := GetLicenseCredentials(license.Spec.LicenseID)
	req.SetBasicAuth(username, password)
}

func GetLicenseCredentials(licenseID string) (string, string) {
	if decoded, err := base64.StdEncoding.DecodeString(licenseID); err == nil {
		var token ServiceAccountToken
		if err := json.Unmarshal(decoded, &token); err == nil && token.Identity != "" && token.Secret != "" {
			return token.Identity, token.Secret
		}
	}

	return licenseID, licenseID
}

func GetLicenseAuthHeader(licenseID string) string {
	username, password := GetLicenseCredentials(licenseID)
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
}
