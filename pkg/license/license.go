package license

import (
	"time"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
)

// LicenseIsExpired checks if a license has expired based on the expires_at entitlement.
// Works with both v1beta1 and v1beta2 licenses via the wrapper.
func LicenseIsExpired(license licensewrapper.LicenseWrapper) (bool, error) {
	// Use wrapper method to get entitlements (works for both v1beta1 and v1beta2)
	entitlements := license.GetEntitlements()

	val, found := entitlements["expires_at"]
	if !found {
		return false, nil
	}
	if val.GetValueType() != "" && val.GetValueType() != "String" {
		return false, errors.Errorf("expires_at must be type String: %s", val.GetValueType())
	}
	if val.GetValue().StrVal == "" {
		return false, nil
	}

	parsed, err := time.Parse(time.RFC3339, val.GetValue().StrVal)
	if err != nil {
		return false, errors.Wrap(err, "failed to parse expiration time")
	}
	return parsed.Before(time.Now()), nil
}

// Deprecated: Use LicenseIsExpired with LicenseWrapper instead.
// This function is maintained for backward compatibility but will be removed in a future version.
func LicenseIsExpiredV1(license *kotsv1beta1.License) (bool, error) {
	wrapper := licensewrapper.LicenseWrapper{V1: license}
	return LicenseIsExpired(wrapper)
}
