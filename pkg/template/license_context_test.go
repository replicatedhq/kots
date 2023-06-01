package template

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/require"
)

func TestLicenseContext_dockercfg(t *testing.T) {
	tests := []struct {
		name          string
		License       *kotsv1beta1.License
		App           *kotsv1beta1.Application
		VersionInfo   *VersionInfo
		expectDecoded map[string]interface{}
	}{
		{
			name: "no app passed to license context should return defaults",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					LicenseID: "abcdef",
				},
			},
			expectDecoded: map[string]interface{}{
				"auths": map[string]interface{}{
					"proxy.replicated.com": map[string]string{
						"auth": base64.StdEncoding.EncodeToString([]byte("abcdef:abcdef")),
					},
					"registry.replicated.com": map[string]string{
						"auth": base64.StdEncoding.EncodeToString([]byte("abcdef:abcdef")),
					},
				},
			},
		},
		{
			name: "app passed with no custom registry domains should return defaults",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					LicenseID: "abcdef",
				},
			},
			App: &kotsv1beta1.Application{
				Spec: kotsv1beta1.ApplicationSpec{
					ProxyRegistryDomain:      "",
					ReplicatedRegistryDomain: "",
				},
			},
			expectDecoded: map[string]interface{}{
				"auths": map[string]interface{}{
					"proxy.replicated.com": map[string]string{
						"auth": base64.StdEncoding.EncodeToString([]byte("abcdef:abcdef")),
					},
					"registry.replicated.com": map[string]string{
						"auth": base64.StdEncoding.EncodeToString([]byte("abcdef:abcdef")),
					},
				},
			},
		},
		{
			name: "app passed with custom registry domains should return custom domains",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					LicenseID: "abcdef",
				},
			},
			App: &kotsv1beta1.Application{
				Spec: kotsv1beta1.ApplicationSpec{
					ProxyRegistryDomain:      "my-proxy.example.com",
					ReplicatedRegistryDomain: "my-registry.example.com",
				},
			},
			expectDecoded: map[string]interface{}{
				"auths": map[string]interface{}{
					"my-proxy.example.com": map[string]string{
						"auth": base64.StdEncoding.EncodeToString([]byte("abcdef:abcdef")),
					},
					"my-registry.example.com": map[string]string{
						"auth": base64.StdEncoding.EncodeToString([]byte("abcdef:abcdef")),
					},
				},
			},
		},
		{
			name: "version info passed with custom domains should return custom domains",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					LicenseID: "abcdef",
				},
			},
			VersionInfo: &VersionInfo{
				ReplicatedProxyDomain:    "my-proxy.example.com",
				ReplicatedRegistryDomain: "my-registry.example.com",
			},
			expectDecoded: map[string]interface{}{
				"auths": map[string]interface{}{
					"my-proxy.example.com": map[string]string{
						"auth": base64.StdEncoding.EncodeToString([]byte("abcdef:abcdef")),
					},
					"my-registry.example.com": map[string]string{
						"auth": base64.StdEncoding.EncodeToString([]byte("abcdef:abcdef")),
					},
				},
			},
		},
		{
			name: "both app and version info are passed with custom domains should return custom domains from version info",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					LicenseID: "abcdef",
				},
			},
			App: &kotsv1beta1.Application{
				Spec: kotsv1beta1.ApplicationSpec{
					ProxyRegistryDomain:      "my-other-proxy.example.com",
					ReplicatedRegistryDomain: "my-other-registry.example.com",
				},
			},
			VersionInfo: &VersionInfo{
				ReplicatedProxyDomain:    "my-proxy.example.com",
				ReplicatedRegistryDomain: "my-registry.example.com",
			},
			expectDecoded: map[string]interface{}{
				"auths": map[string]interface{}{
					"my-proxy.example.com": map[string]string{
						"auth": base64.StdEncoding.EncodeToString([]byte("abcdef:abcdef")),
					},
					"my-registry.example.com": map[string]string{
						"auth": base64.StdEncoding.EncodeToString([]byte("abcdef:abcdef")),
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			ctx := licenseCtx{License: test.License, App: test.App, VersionInfo: test.VersionInfo}

			expectJson, err := json.Marshal(test.expectDecoded)
			req.NoError(err)

			expect := base64.StdEncoding.EncodeToString(expectJson)

			dockercfg := ctx.licenseDockercfg()
			req.Equal(expect, dockercfg)
		})
	}
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
			name: "built-in isSnapshotSupported",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					IsSnapshotSupported: true,
				},
			},
			fieldName: "isSnapshotSupported",
			want:      "true",
		},
		{
			name: "built-in isGeoaxisSupported",
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					IsGeoaxisSupported: true,
				},
			},
			fieldName: "isGeoaxisSupported",
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
