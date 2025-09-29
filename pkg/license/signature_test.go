package license

import (
	"embed"
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsutil"
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
