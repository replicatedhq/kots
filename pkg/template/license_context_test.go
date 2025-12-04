package template

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
	"github.com/stretchr/testify/require"
)

func TestLicenseContext_dockercfg(t *testing.T) {
	tests := []struct {
		name          string
		License       licensewrapper.LicenseWrapper
		App           *kotsv1beta1.Application
		VersionInfo   *VersionInfo
		expectDecoded map[string]interface{}
	}{
		{
			name: "no app passed to license context should return defaults",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						LicenseID: "abcdef",
					},
				},
			},
			expectDecoded: map[string]interface{}{
				"auths": map[string]interface{}{
					"proxy.replicated.com": map[string]string{
						"auth": base64.StdEncoding.EncodeToString([]byte("LICENSE_ID:abcdef")),
					},
					"registry.replicated.com": map[string]string{
						"auth": base64.StdEncoding.EncodeToString([]byte("LICENSE_ID:abcdef")),
					},
				},
			},
		},
		{
			name: "app passed with no custom registry domains should return defaults",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						LicenseID: "abcdef",
					},
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
						"auth": base64.StdEncoding.EncodeToString([]byte("LICENSE_ID:abcdef")),
					},
					"registry.replicated.com": map[string]string{
						"auth": base64.StdEncoding.EncodeToString([]byte("LICENSE_ID:abcdef")),
					},
				},
			},
		},
		{
			name: "app passed with custom registry domains should return custom domains",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						LicenseID: "abcdef",
					},
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
						"auth": base64.StdEncoding.EncodeToString([]byte("LICENSE_ID:abcdef")),
					},
					"my-registry.example.com": map[string]string{
						"auth": base64.StdEncoding.EncodeToString([]byte("LICENSE_ID:abcdef")),
					},
				},
			},
		},
		{
			name: "version info passed with custom domains should return custom domains",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						LicenseID: "abcdef",
					},
				},
			},
			VersionInfo: &VersionInfo{
				ReplicatedProxyDomain:    "my-proxy.example.com",
				ReplicatedRegistryDomain: "my-registry.example.com",
			},
			expectDecoded: map[string]interface{}{
				"auths": map[string]interface{}{
					"my-proxy.example.com": map[string]string{
						"auth": base64.StdEncoding.EncodeToString([]byte("LICENSE_ID:abcdef")),
					},
					"my-registry.example.com": map[string]string{
						"auth": base64.StdEncoding.EncodeToString([]byte("LICENSE_ID:abcdef")),
					},
				},
			},
		},
		{
			name: "both app and version info are passed with custom domains should return custom domains from version info",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						LicenseID: "abcdef",
					},
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
						"auth": base64.StdEncoding.EncodeToString([]byte("LICENSE_ID:abcdef")),
					},
					"my-registry.example.com": map[string]string{
						"auth": base64.StdEncoding.EncodeToString([]byte("LICENSE_ID:abcdef")),
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			ctx := licenseCtx{License: &test.License, App: test.App, VersionInfo: test.VersionInfo}

			expectJson, err := json.Marshal(test.expectDecoded)
			req.NoError(err)

			expect := base64.StdEncoding.EncodeToString(expectJson)

			dockercfg, err := ctx.licenseDockercfg()
			req.NoError(err)
			req.Equal(expect, dockercfg)
		})
	}
}

func TestLicenseCtx_licenseFieldValue(t *testing.T) {
	tests := []struct {
		name      string
		License   licensewrapper.LicenseWrapper
		fieldName string
		want      string
	}{
		{
			name:      "license is nil",
			License:   licensewrapper.LicenseWrapper{},
			fieldName: "doesNotExist",
			want:      "",
		},
		{
			name: "licenseField does not exist",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
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
			},
			fieldName: "doesNotExist",
			want:      "",
		},
		{
			name: "exists as integer",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
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
			},
			fieldName: "integerField",
			want:      "587",
		},
		{
			name: "exists as string",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
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
			},
			fieldName: "strField",
			want:      "strValue",
		},
		{
			name: "built-in isGitOpsSupported",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
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
			},
			fieldName: "isGitOpsSupported",
			want:      "true",
		},
		{
			name: "built-in isEmbeddedClusterMultiNodeEnabled",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsEmbeddedClusterMultiNodeEnabled: true,
					},
				},
			},
			fieldName: "isEmbeddedClusterMultiNodeEnabled",
			want:      "true",
		},
		{
			name: "built-in isIdentityServiceSupported",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsIdentityServiceSupported: true,
					},
				},
			},
			fieldName: "isIdentityServiceSupported",
			want:      "true",
		},
		{
			name: "built-in isSnapshotSupported",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsSnapshotSupported: true,
					},
				},
			},
			fieldName: "isSnapshotSupported",
			want:      "true",
		},
		{
			name: "built-in IsDisasterRecoverySupported",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsDisasterRecoverySupported: true,
					},
				},
			},
			fieldName: "IsDisasterRecoverySupported",
			want:      "true",
		},
		{
			name: "built-in isGeoaxisSupported",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsGeoaxisSupported: true,
					},
				},
			},
			fieldName: "isGeoaxisSupported",
			want:      "true",
		},
		{
			name: "built-in isAirgapSupported",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsAirgapSupported: true,
					},
				},
			},
			fieldName: "isAirgapSupported",
			want:      "true",
		},
		{
			name: "built-in licenseSequence",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						LicenseSequence: 987,
					},
				},
			},
			fieldName: "licenseSequence",
			want:      "987",
		},
		{
			name: "built-in licenseType",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						LicenseType: "test",
					},
				},
			},
			fieldName: "licenseType",
			want:      "test",
		},
		{
			name: "built-in appSlug",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						AppSlug: "appSlug",
					},
				},
			},
			fieldName: "appSlug",
			want:      "appSlug",
		},
		{
			name: "built-in channelName",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						ChannelName: "stable",
					},
				},
			},
			fieldName: "channelName",
			want:      "stable",
		},
		{
			name: "built-in customerID",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						CustomerID: "123",
					},
				},
			},
			fieldName: "customerID",
			want:      "123",
		},
		{
			name: "built-in customerName",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						CustomerName: "name",
					},
				},
			},
			fieldName: "customerName",
			want:      "name",
		},
		{
			name: "built-in licenseID",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						LicenseID: "123",
					},
				},
			},
			fieldName: "licenseID",
			want:      "123",
		},
		{
			name: "built-in licenseId",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						LicenseID: "123",
					},
				},
			},
			fieldName: "licenseId",
			want:      "123",
		},
		{
			name: "built-in signature",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
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
			},
			fieldName: "signature",
			want:      "abcdef0123456789",
		},
		{
			name: "built-in signature with a custom field of the same name",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
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
			},
			fieldName: "signature",
			want:      "abcdef0123456789",
		},
		{
			name: "built-in channelID",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						ChannelID: "channel-123",
					},
				},
			},
			fieldName: "channelID",
			want:      "channel-123",
		},
		{
			name: "built-in customerEmail",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						CustomerEmail: "customer@example.com",
					},
				},
			},
			fieldName: "customerEmail",
			want:      "customer@example.com",
		},
		{
			name: "built-in endpoint",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						Endpoint: "https://replicated.example.com",
					},
				},
			},
			fieldName: "endpoint",
			want:      "https://replicated.example.com",
		},
		{
			name: "built-in endpoint with empty value returns default",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						Endpoint: "",
					},
				},
			},
			fieldName: "endpoint",
			want:      "https://replicated.app",
		},
		{
			name: "built-in replicatedProxyDomain",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						ReplicatedProxyDomain: "proxy.example.com",
					},
				},
			},
			fieldName: "replicatedProxyDomain",
			want:      "proxy.example.com",
		},
		{
			name: "built-in isEmbeddedClusterDownloadEnabled",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsEmbeddedClusterDownloadEnabled: true,
					},
				},
			},
			fieldName: "isEmbeddedClusterDownloadEnabled",
			want:      "true",
		},
		{
			name: "built-in isSemverRequired",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsSemverRequired: true,
					},
				},
			},
			fieldName: "isSemverRequired",
			want:      "true",
		},
		{
			name: "built-in isSupportBundleUploadSupported",
			License: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsSupportBundleUploadSupported: true,
					},
				},
			},
			fieldName: "isSupportBundleUploadSupported",
			want:      "true",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			ctx := licenseCtx{
				License: &tt.License,
			}
			req.Equal(tt.want, ctx.licenseFieldValue(tt.fieldName))
		})
	}
}
