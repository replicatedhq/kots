package template

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/require"
)

func TestLicenseContext_dockercfg(t *testing.T) {
	req := require.New(t)

	ctx := licenseCtx{
		License: &kotsv1beta1.License{
			Spec: kotsv1beta1.LicenseSpec{
				LicenseID: "abcdef",
			},
		},
	}

	expect := "eyJhdXRocyI6eyJwcm94eS5yZXBsaWNhdGVkLmNvbSI6eyJhdXRoIjoiWVdKalpHVm1PbUZpWTJSbFpnPT0ifSwicmVnaXN0cnkucmVwbGljYXRlZC5jb20iOnsiYXV0aCI6IllXSmpaR1ZtT21GaVkyUmxaZz09In19fQ=="
	dockercfg := ctx.licenseDockercfg()
	req.Equal(expect, dockercfg)
}

func TestLicenseCtx_licenseFieldValue(t *testing.T) {
	tests := []struct {
		name      string
		License   *kotsv1beta1.License
		fieldName string
		want      string
	}{
		{
			name:      "license is nil",
			License:   nil,
			fieldName: "doesNotExist",
			want:      "",
		},
		{
			name: "licenseField does not exist",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					Entitlements: map[string]kotsv1beta1.EntitlementField{
						"abc": {
							Value: kotsv1beta1.EntitlementValue{
								Type:   kotsv1beta1.String,
								StrVal: "abc",
							},
						},
					},
				},
			},
			fieldName: "doesNotExist",
			want:      "",
		},
		{
			name: "exists as integer",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					Entitlements: map[string]kotsv1beta1.EntitlementField{
						"integerField": {
							Value: kotsv1beta1.EntitlementValue{
								Type:   kotsv1beta1.Int,
								IntVal: 587,
							},
						},
					},
				},
			},
			fieldName: "integerField",
			want:      "587",
		},
		{
			name: "exists as string",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					Entitlements: map[string]kotsv1beta1.EntitlementField{
						"strField": {
							Value: kotsv1beta1.EntitlementValue{
								Type:   kotsv1beta1.String,
								StrVal: "strValue",
							},
						},
					},
				},
			},
			fieldName: "strField",
			want:      "strValue",
		},
		{
			name: "built-in isGitOpsSupported",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					IsGitOpsSupported: true,
					Entitlements: map[string]kotsv1beta1.EntitlementField{
						"strField": {
							Value: kotsv1beta1.EntitlementValue{
								Type:   kotsv1beta1.String,
								StrVal: "strValue",
							},
						},
					},
				},
			},
			fieldName: "isGitOpsSupported",
			want:      "true",
		},
		{
			name: "built-in isIdentityServiceSupported",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					IsIdentityServiceSupported: true,
				},
			},
			fieldName: "isIdentityServiceSupported",
			want:      "true",
		},
		{
			name: "built-in isAirgapSupported",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					IsAirgapSupported: true,
				},
			},
			fieldName: "isAirgapSupported",
			want:      "true",
		},
		{
			name: "built-in licenseSequence",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					LicenseSequence: 987,
				},
			},
			fieldName: "licenseSequence",
			want:      "987",
		},
		{
			name: "built-in licenseType",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					LicenseType: "test",
				},
			},
			fieldName: "licenseType",
			want:      "test",
		},
		{
			name: "built-in appSlug",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					AppSlug: "appSlug",
				},
			},
			fieldName: "appSlug",
			want:      "appSlug",
		},
		{
			name: "built-in channelName",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					ChannelName: "stable",
				},
			},
			fieldName: "channelName",
			want:      "stable",
		},
		{
			name: "built-in customerName",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					CustomerName: "name",
				},
			},
			fieldName: "customerName",
			want:      "name",
		},
		{
			name: "built-in licenseID",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					LicenseID: "123",
				},
			},
			fieldName: "licenseID",
			want:      "123",
		},
		{
			name: "built-in licenseId",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					LicenseID: "123",
				},
			},
			fieldName: "licenseId",
			want:      "123",
		},
		{
			name: "built-in signature",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					IsGitOpsSupported: true,
					Entitlements: map[string]kotsv1beta1.EntitlementField{
						"strField": {
							Value: kotsv1beta1.EntitlementValue{
								Type:   kotsv1beta1.String,
								StrVal: "strValue",
							},
						},
					},
					Signature: []byte("abcdef0123456789"),
				},
			},
			fieldName: "signature",
			want:      "abcdef0123456789",
		},
		{
			name: "built-in signature with a custom field of the same name",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					IsGitOpsSupported: true,
					Entitlements: map[string]kotsv1beta1.EntitlementField{
						"signature": {
							Value: kotsv1beta1.EntitlementValue{
								Type:   kotsv1beta1.String,
								StrVal: "strValue",
							},
						},
					},
					Signature: []byte("abcdef0123456789"),
				},
			},
			fieldName: "signature",
			want:      "abcdef0123456789",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			ctx := licenseCtx{
				License: tt.License,
			}
			req.Equal(tt.want, ctx.licenseFieldValue(tt.fieldName))
		})
	}
}
