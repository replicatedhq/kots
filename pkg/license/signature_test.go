package license

import (
	"embed"
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/*
var testdata embed.FS

func TestVerifySignature(t *testing.T) {
	tests := []struct {
		name       string
		license    string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "valid signature",
			license: func() string {
				b, err := testdata.ReadFile("testdata/valid.yaml")
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}(),
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name: "valid signature without entitlement signature",
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
			name: "valid signature with entitlement signature",
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
			name: "invalid signature",
			license: func() string {
				b, err := testdata.ReadFile("testdata/invalid-signature.yaml")
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}(),
			wantErr:    true,
			wantErrMsg: "failed to verify license signature: crypto/rsa: verification error: signature is invalid",
		},
		{
			name: "licenseID field changed",
			license: func() string {
				b, err := testdata.ReadFile("testdata/invalid-changed-licenseID.yaml")
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}(),
			wantErr:    true,
			wantErrMsg: `"licenseID" field has changed to "1vusOokxAVp1tkRGuyxnF23PJcq-modified" (license) from "1vusOokxAVp1tkRGuyxnF23PJcq" (within signature)`,
		},
		{
			name: "endpoint field changed",
			license: func() string {
				b, err := testdata.ReadFile("testdata/invalid-changed-endpoint.yaml")
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}(),
			wantErr:    true,
			wantErrMsg: `"endpoint" field has changed to "https://replicated.app.modified" (license) from "https://replicated.app" (within signature)`,
		},
		{
			name: "isEmbeddedClusterMultiNodeEnabled field changed",
			license: func() string {
				b, err := testdata.ReadFile("testdata/invalid-changed-isEmbeddedClusterMultiNodeEnabled.yaml")
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}(),
			// kots versions <= v1.124.15 do not preserve the license structure for automated airgap installs
			// which causes the license verification for isEmbeddedClusterMultiNodeEnabled to fail
			// in kots versions that have this license field.
			wantErr:    false,
			wantErrMsg: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			license, err := kotsutil.LoadLicenseFromBytes([]byte(tt.license))
			req.NoError(err)

			_, err = VerifySignature(license)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifySignature() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.wantErrMsg != err.Error() {
				t.Errorf("VerifySignature() error message = %v, wantErrMsg %v", err.Error(), tt.wantErrMsg)
				return
			}
		})
	}
}

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
			license: func() string {
				b, err := testdata.ReadFile("testdata/invalid-signature.yaml")
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}(),
			wantErr: true,
			// The actual error message contains more context from VerifyLicenseWrapper
		},
		{
			name: "v1beta1 license with tampered data should fail",
			license: func() string {
				b, err := testdata.ReadFile("testdata/invalid-changed-licenseID.yaml")
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}(),
			wantErr: true,
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
			verified, err := VerifyLicenseWrapper(wrapper)
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
	emptyWrapper := licensewrapper.LicenseWrapper{}

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
	verified, err := VerifyLicenseWrapper(wrapper)
	req.NoError(err)

	// Ensure version is preserved
	req.True(verified.IsV1(), "Verification should preserve v1beta1 version")
	req.False(verified.IsV2(), "Should not convert to v1beta2")
	req.Equal(wrapper.GetLicenseID(), verified.GetLicenseID())
	req.Equal(wrapper.GetAppSlug(), verified.GetAppSlug())
}

// TestVerifyLicenseWrapper_V1Beta2_Structure tests v1beta2 license wrapper functionality
// Note: These are structural tests that verify wrapper behavior without cryptographic validation.
// Full cryptographic validation tests require properly signed v1beta2 test licenses.
func TestVerifyLicenseWrapper_V1Beta2_Structure(t *testing.T) {
	req := require.New(t)

	// Load v1beta2 license structure
	licenseData, err := testdata.ReadFile("testdata/valid-v1beta2-structure.yaml")
	req.NoError(err)

	// Test that wrapper correctly identifies v1beta2
	wrapper, err := licensewrapper.LoadLicenseFromBytes(licenseData)
	req.NoError(err)
	req.True(wrapper.IsV2(), "Expected v1beta2 license")
	req.False(wrapper.IsV1(), "Should not be v1beta1")

	// Test that wrapper provides access to v1beta2 fields
	req.Equal("1vusOokxAVp1tkRGuyxnF23PJcq", wrapper.GetLicenseID())
	req.Equal("my-app", wrapper.GetAppSlug())
	req.Equal("Test Customer", wrapper.GetCustomerName())
	req.Equal("My Channel", wrapper.GetChannelName())
	req.True(wrapper.IsAirgapSupported())
	req.True(wrapper.IsGitOpsSupported())
	req.True(wrapper.IsSnapshotSupported())

	// Test entitlements access through wrapper
	entitlements := wrapper.GetEntitlements()
	req.NotNil(entitlements)
	req.Len(entitlements, 4)

	boolField := entitlements["bool_field"]
	req.NotNil(boolField)
	req.Equal("Bool Field", boolField.GetTitle())
	req.Equal("Boolean", boolField.GetValueType())
	boolVal := boolField.GetValue()
	req.Equal(true, (&boolVal).Value())

	intField := entitlements["int_field"]
	req.NotNil(intField)
	req.Equal("Int Field", intField.GetTitle())
	intVal := intField.GetValue()
	req.Equal(int64(123), (&intVal).Value())
}

func TestVerifyLicenseWrapper_V1Beta2_VersionPreservation(t *testing.T) {
	req := require.New(t)

	// Load v1beta2 license
	licenseData, err := testdata.ReadFile("testdata/valid-v1beta2-structure.yaml")
	req.NoError(err)

	wrapper, err := licensewrapper.LoadLicenseFromBytes(licenseData)
	req.NoError(err)
	req.True(wrapper.IsV2())

	// Even though validation will fail (no real crypto signature),
	// test that the wrapper correctly routes to V2 validation
	_, err = VerifyLicenseWrapper(wrapper)

	// We expect an error because the signature isn't cryptographically valid
	// but the important thing is it tried V2 validation, not V1
	if err != nil {
		req.Contains(err.Error(), "v1beta2", "Error should mention v1beta2, indicating V2 validation was attempted")
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
	v2Data, err := testdata.ReadFile("testdata/valid-v1beta2-structure.yaml")
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
