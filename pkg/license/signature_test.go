package license

import (
	"embed"
	"testing"

	"github.com/replicatedhq/kotskinds/pkg/crypto"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/*
var testdata embed.FS

// TestVerifyLicenseWrapper_V1Beta1 tests v1beta1 license verification with MD5 signatures.
// It verifies two distinct error paths:
// 1. Cryptographic signature failures (invalid/tampered signatures) -> returns error
// 2. Data validation errors (field mismatches between outer and signed license) -> logs warning, returns success
func TestVerifyLicenseWrapper_V1Beta1(t *testing.T) {
	tests := []struct {
		name       string
		license    string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "valid v1beta1 license with MD5 signature (no entitlement signatures)",
			license: func() string {
				b, err := testdata.ReadFile("testdata/valid-without-entitlement-signature.yaml")
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}(),
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name: "valid v1beta1 license with entitlement signature",
			license: func() string {
				b, err := testdata.ReadFile("testdata/valid-with-entitlement-signature.yaml")
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}(),
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name: "invalid v1beta1 signature should fail",
			// Cryptographic signature is invalid - this should return an error
			license: func() string {
				b, err := testdata.ReadFile("testdata/invalid-signature.yaml")
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}(),
			wantErr: true,
		},
		{
			name: "v1beta1 license with isEmbeddedClusterMultiNodeEnabled field mismatch logs warning but succeeds",
			// Data validation error: field value differs between outer license and signed inner license.
			// The cryptographic signature is valid, so this logs a warning but succeeds.
			license: func() string {
				b, err := testdata.ReadFile("testdata/invalid-changed-isEmbeddedClusterMultiNodeEnabled.yaml")
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			// Load license using wrapper (auto-detects v1beta1)
			wrapper, err := licensewrapper.LoadLicenseFromBytes([]byte(tt.license))
			req.NoError(err)
			req.True(wrapper.IsV1(), "Expected v1beta1 license")

			// Verify using wrapper function
			verified, err := VerifyLicenseWrapper(&wrapper)
			if tt.wantErr {
				req.Error(err)
				if tt.wantErrMsg != "" {
					req.Contains(err.Error(), tt.wantErrMsg)
				}
				return
			}

			req.NoError(err)
			req.True(verified.IsV1(), "Expected verified license to be v1beta1")
			req.Equal(wrapper.GetLicenseID(), verified.GetLicenseID())
		})
	}
}

func TestVerifyLicenseWrapper_EmptyWrapper(t *testing.T) {
	req := require.New(t)

	// Test empty wrapper (neither V1 nor V2)
	var emptyWrapper *licensewrapper.LicenseWrapper = nil
	_, err := VerifyLicenseWrapper(emptyWrapper)
	req.Error(err)
	req.Contains(err.Error(), "license wrapper contains no license")
}

func TestVerifyLicenseWrapper_PreservesVersion(t *testing.T) {
	req := require.New(t)

	// Load a valid v1beta1 license (without entitlement signatures)
	licenseData, err := testdata.ReadFile("testdata/valid-without-entitlement-signature.yaml")
	req.NoError(err)

	wrapper, err := licensewrapper.LoadLicenseFromBytes(licenseData)
	req.NoError(err)
	req.True(wrapper.IsV1())

	// Verify it
	verified, err := VerifyLicenseWrapper(&wrapper)
	req.NoError(err)

	// Ensure version is preserved
	req.True(verified.IsV1(), "Verification should preserve v1beta1 version")
	req.False(verified.IsV2(), "Should not convert to v1beta2")
	req.Equal(wrapper.GetLicenseID(), verified.GetLicenseID())
	req.Equal(wrapper.GetAppSlug(), verified.GetAppSlug())
}

// TestVerifyLicenseWrapper_V1Beta2 tests v1beta2 license verification with SHA-256 signatures.
// It verifies two distinct error paths:
// 1. Cryptographic signature failures (invalid/tampered signatures) -> returns error
// 2. Data validation errors (field mismatches between outer and signed license) -> logs warning, returns success
func TestVerifyLicenseWrapper_V1Beta2(t *testing.T) {
	// Set up custom global key for v1beta2 test licenses
	// This key was used to sign the test licenses in testdata/
	globalKey := `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxHh2OXzDqlQ7kZJ1d4zr
wbpXsSFHcYzr+k6pe+QXLUelAMvlik9NXauIt+YFtEAxNypV+xPCr8ClH5L2qPPb
QBeG0ExxzvRshDMGxm7TXVHzTXQCrD7azS8Va6RsAB4tJMlvymn2uHsQDbShQiOY
RKaRY/KKBmaIcYmysaSvfU8E5Ve9f4478X3u1cPzKUG6dk5j1Nt3nSv3BWINM5ec
IXJQCB+gQVkOjzvA9aRVtLJtFqAoX7A6BfTNqrx35eyBEmzQOo0Mx1JkZDDW4+qC
bhC0kq14IRpwKFIALBhSojfbJelM+gCv3wjF4hrWxAZQzWSPexP1Msof2KbrniEe
LQIDAQAB
-----END PUBLIC KEY-----
`
	err := crypto.SetCustomPublicKeyRSA(globalKey)
	require.NoError(t, err, "failed to set custom global key for v1beta2 tests")

	tests := []struct {
		name       string
		license    string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "valid v1beta2 license with SHA-256 signature",
			license: func() string {
				b, err := testdata.ReadFile("testdata/valid-v1beta2.yaml")
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}(),
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name: "invalid v1beta2 signature should fail",
			// Cryptographic signature is invalid - this should return an error
			license: func() string {
				b, err := testdata.ReadFile("testdata/invalid-v1beta2-signature.yaml")
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}(),
			wantErr: true,
		},
		{
			name: "v1beta2 license with licenseID field mismatch logs warning but succeeds",
			// Data validation error: licenseID field differs between outer license and signed inner license.
			// The cryptographic signature is valid, so this logs a warning but succeeds.
			license: func() string {
				b, err := testdata.ReadFile("testdata/invalid-v1beta2-changed-licenseID.yaml")
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			// Load license using wrapper (auto-detects v1beta2)
			wrapper, err := licensewrapper.LoadLicenseFromBytes([]byte(tt.license))
			req.NoError(err)
			req.True(wrapper.IsV2(), "Expected v1beta2 license")

			// Verify using wrapper function
			verified, err := VerifyLicenseWrapper(&wrapper)
			if tt.wantErr {
				req.Error(err)
				if tt.wantErrMsg != "" {
					req.Contains(err.Error(), tt.wantErrMsg)
				}
				return
			}

			req.NoError(err)
			req.True(verified.IsV2(), "Expected verified license to be v1beta2")
			req.Equal(wrapper.GetLicenseID(), verified.GetLicenseID())
		})
	}
}

func TestVerifyLicenseWrapper_MixedVersion(t *testing.T) {
	req := require.New(t)

	// Test v1beta1
	v1Data, err := testdata.ReadFile("testdata/valid-without-entitlement-signature.yaml")
	req.NoError(err)
	v1Wrapper, err := licensewrapper.LoadLicenseFromBytes(v1Data)
	req.NoError(err)
	req.True(v1Wrapper.IsV1())
	req.False(v1Wrapper.IsV2())

	// Test v1beta2
	v2Data, err := testdata.ReadFile("testdata/valid-v1beta2.yaml")
	req.NoError(err)
	v2Wrapper, err := licensewrapper.LoadLicenseFromBytes(v2Data)
	req.NoError(err)
	req.False(v2Wrapper.IsV1())
	req.True(v2Wrapper.IsV2())

	// Verify both wrappers provide access to logical fields through wrapper methods
	// (values may differ between test fixtures, but methods should work for both versions)
	req.NotEqual(v1Wrapper.GetLicenseID(), "")
	req.NotEqual(v2Wrapper.GetLicenseID(), "")
	req.NotEqual(v1Wrapper.GetAppSlug(), "")
	req.NotEqual(v2Wrapper.GetAppSlug(), "")
	req.NotEqual(v1Wrapper.GetCustomerName(), "")
	req.NotEqual(v2Wrapper.GetCustomerName(), "")
}
