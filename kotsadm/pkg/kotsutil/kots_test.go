package kotsutil

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.undefinedlabs.com/scopeagent"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_LoadLicenseFromBytes(t *testing.T) {
	tests := []struct {
		name                    string
		data                    []byte
		expectedLicense         *kotsv1beta1.License
		expectedUnsignedLicense *kotsv1beta1.UnsignedLicense
	}{
		{
			name: "license",
			data: []byte(`apiVersion: kots.io/v1beta1
kind: License
metadata:
  name: license
spec:
  appSlug: slug
  channelName: Beta
  customerName: Customer
  endpoint: https://replicated.app
  entitlements:
    expires_at:
      description: License Expiration
      title: Expiration
      value: ""
      valueType: String
  isGitOpsSupported: true
  isSnapshotSupported: true
  licenseID: abcdef
  licenseSequence: 3
  licenseType: dev
  signature: c2lnbmVk`),
			expectedLicense: &kotsv1beta1.License{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kots.io/v1beta1",
					Kind:       "License",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "license",
				},
				Spec: kotsv1beta1.LicenseSpec{
					AppSlug:      "slug",
					ChannelName:  "Beta",
					CustomerName: "Customer",
					Endpoint:     "https://replicated.app",
					Entitlements: map[string]kotsv1beta1.EntitlementField{
						"expires_at": {
							Description: "License Expiration",
							Title:       "Expiration",
							Value: kotsv1beta1.EntitlementValue{
								Type:   kotsv1beta1.String,
								StrVal: "",
							},
							ValueType: "String",
						},
					},
					IsGitOpsSupported:   true,
					IsSnapshotSupported: true,
					LicenseID:           "abcdef",
					LicenseSequence:     3,
					LicenseType:         "dev",
					Signature:           []byte("signed"),
				},
			},
		},
		{
			name: "unsigned license",
			data: []byte(`apiVersion: kots.io/v1beta1
kind: UnsignedLicense
metadata:
  name: local
spec:
  endpoint: http://kots-server:3000/chaos-mesh
  slug: chaos-mesh`),
			expectedUnsignedLicense: &kotsv1beta1.UnsignedLicense{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kots.io/v1beta1",
					Kind:       "UnsignedLicense",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "local",
				},
				Spec: kotsv1beta1.UnsignedLicenseSpec{
					Endpoint: "http://kots-server:3000/chaos-mesh",
					Slug:     "chaos-mesh",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scopetest := scopeagent.StartTest(t)
			defer scopetest.End()

			req := require.New(t)

			actualLicense, actualUnsignedLicense, err := LoadLicenseFromBytes(test.data)
			req.NoError(err)

			assert.Equal(t, test.expectedLicense, actualLicense)
			assert.Equal(t, test.expectedUnsignedLicense, actualUnsignedLicense)
		})
	}
}
