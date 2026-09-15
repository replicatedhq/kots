package license

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsv1beta2 "github.com/replicatedhq/kotskinds/apis/kots/v1beta2"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
	"github.com/stretchr/testify/require"
)

func TestValidateCustomerID(t *testing.T) {
	v1 := func(id string) *licensewrapper.LicenseWrapper {
		return &licensewrapper.LicenseWrapper{V1: &kotsv1beta1.License{Spec: kotsv1beta1.LicenseSpec{CustomerID: id}}}
	}
	v2 := func(id string) *licensewrapper.LicenseWrapper {
		return &licensewrapper.LicenseWrapper{V2: &kotsv1beta2.License{Spec: kotsv1beta2.LicenseSpec{CustomerID: id}}}
	}
	for _, tc := range []struct {
		name    string
		current *licensewrapper.LicenseWrapper
		updated *licensewrapper.LicenseWrapper
		wantErr bool
	}{
		{"unchanged v1 to v2", v1("customer-a"), v2("customer-a"), false},
		{"changed", v1("customer-a"), v2("customer-b"), true},
		{"missing new ID uses fallback", v2("customer-a"), v1(""), false},
		{"legacy receives ID", v1(""), v2("customer-a"), false},
		{"legacy stays empty", v1(""), v1(""), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCustomerID(tc.current, tc.updated)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateLicenseIdentity(t *testing.T) {
	license := func(customerID, licenseID string) *licensewrapper.LicenseWrapper {
		return &licensewrapper.LicenseWrapper{V1: &kotsv1beta1.License{Spec: kotsv1beta1.LicenseSpec{
			CustomerID: customerID,
			LicenseID:  licenseID,
		}}}
	}
	for _, tc := range []struct {
		name    string
		current *licensewrapper.LicenseWrapper
		updated *licensewrapper.LicenseWrapper
		wantErr bool
	}{
		{"same customer permits new license ID", license("customer-a", "license-a"), license("customer-a", "license-b"), false},
		{"different customer is rejected", license("customer-a", "license-a"), license("customer-b", "license-a"), true},
		{"legacy same license ID", license("", "license-a"), license("customer-a", "license-a"), false},
		{"legacy changed license ID", license("", "license-a"), license("customer-a", "license-b"), true},
		{"both IDs missing same license ID", license("", "license-a"), license("", "license-a"), false},
		{"both IDs missing changed license ID", license("", "license-a"), license("", "license-b"), true},
		{"new ID missing same license ID", license("customer-a", "license-a"), license("", "license-a"), false},
		{"new ID missing changed license ID", license("customer-a", "license-a"), license("", "license-b"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateLicenseIdentity(tc.current, tc.updated)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRejectSameLicenseForChange(t *testing.T) {
	license := func(customerID, licenseID string) *licensewrapper.LicenseWrapper {
		return &licensewrapper.LicenseWrapper{V1: &kotsv1beta1.License{Spec: kotsv1beta1.LicenseSpec{
			CustomerID: customerID,
			LicenseID:  licenseID,
		}}}
	}
	for _, tc := range []struct {
		name    string
		current *licensewrapper.LicenseWrapper
		new     *licensewrapper.LicenseWrapper
		wantErr bool
	}{
		{"same customer with new license ID", license("customer-a", "license-a"), license("customer-a", "license-b"), true},
		{"different customer with same license ID", license("customer-a", "license-a"), license("customer-b", "license-a"), false},
		{"legacy same license ID", license("", "license-a"), license("customer-b", "license-a"), true},
		{"legacy new license ID", license("", "license-a"), license("customer-b", "license-b"), false},
		{"new customer ID missing, same license ID", license("customer-a", "license-a"), license("", "license-a"), true},
		{"new customer ID missing, new license ID", license("customer-a", "license-a"), license("", "license-b"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := RejectSameLicenseForChange(tc.current, tc.new)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
