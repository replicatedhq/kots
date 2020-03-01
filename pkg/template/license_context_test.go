package template

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/require"
	"go.undefinedlabs.com/scopeagent"
)

func TestLicenseContext_dockercfg(t *testing.T) {
	scopetest := scopeagent.StartTest(t)
	defer scopetest.End()
	req := require.New(t)

	ctx := LicenseCtx{
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
						"abc": kotsv1beta1.EntitlementField{
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
						"integerField": kotsv1beta1.EntitlementField{
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
						"strField": kotsv1beta1.EntitlementField{
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scopetest := scopeagent.StartTest(t)
			defer scopetest.End()
			req := require.New(t)

			ctx := LicenseCtx{
				License: tt.License,
			}
			req.Equal(tt.want, ctx.licenseFieldValue(tt.fieldName))
		})
	}
}
