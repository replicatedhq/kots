package license

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
)

// ValidateCustomerID compares customer IDs when both licenses have one.
// Older licenses without the field use license ID validation instead.
func ValidateCustomerID(current, updated *licensewrapper.LicenseWrapper) error {
	if updated == nil || updated.IsEmpty() {
		return errors.New("new license is required")
	}
	if current != nil && !current.IsEmpty() && current.GetCustomerID() != "" && updated.GetCustomerID() != "" && current.GetCustomerID() != updated.GetCustomerID() {
		return errors.Errorf("customer IDs do not match: current %q, new %q", current.GetCustomerID(), updated.GetCustomerID())
	}
	return nil
}

// ValidateLicenseIdentity falls back to license ID (or matching service account
// identity) when either license is missing a customer ID.
func ValidateLicenseIdentity(current, updated *licensewrapper.LicenseWrapper) error {
	if err := ValidateCustomerID(current, updated); err != nil {
		return err
	}
	if current == nil || current.IsEmpty() {
		return errors.New("current license is required")
	}
	if current.GetCustomerID() != "" && updated.GetCustomerID() != "" {
		return nil
	}
	if current.GetLicenseID() == updated.GetLicenseID() {
		return nil
	}
	if _, _, err := ValidateServiceAccountToken(updated.GetLicenseID(), current); err == nil {
		return nil
	}
	return errors.Errorf("license IDs do not match: current %q, new %q", current.GetLicenseID(), updated.GetLicenseID())
}
