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

// TestVerifyLicenseWrapper_V1Beta2 tests v1beta2 license validation
// Note: This test requires actual v1beta2 license test data with SHA-256 signatures
// to be added to testdata/ directory in the future
func TestVerifyLicenseWrapper_V1Beta2(t *testing.T) {
	t.Skip("Skipping v1beta2 test - requires v1beta2 test data with SHA-256 signatures")

	// TODO: Add v1beta2 test data files to testdata/:
	// - testdata/valid-v1beta2.yaml (with valid SHA-256 signature)
	// - testdata/invalid-v1beta2-signature.yaml (with invalid SHA-256)
	// - testdata/invalid-v1beta2-tampered.yaml (with tampered data)
	//
	// Test structure:
	// 1. Load v1beta2 license using licensewrapper.LoadLicenseFromBytes()
	// 2. Verify wrapper.IsV2() returns true
	// 3. Call VerifyLicenseWrapper(wrapper)
	// 4. Assert validation succeeds for valid licenses
	// 5. Assert validation fails for invalid signatures
	// 6. Assert version is preserved after validation
}
