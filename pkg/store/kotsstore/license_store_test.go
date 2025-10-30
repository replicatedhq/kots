package kotsstore

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsv1beta2 "github.com/replicatedhq/kotskinds/apis/kots/v1beta2"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestLicenseWrapper_V1Beta2_Detection tests that the wrapper correctly identifies v1beta2 licenses
func TestLicenseWrapper_V1Beta2_Detection(t *testing.T) {
	tests := []struct {
		name          string
		wrapper       licensewrapper.LicenseWrapper
		expectIsV1    bool
		expectIsV2    bool
		expectAppSlug string
	}{
		{
			name: "v1beta1 license",
			wrapper: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						AppSlug: "test-app-v1",
					},
				},
			},
			expectIsV1:    true,
			expectIsV2:    false,
			expectAppSlug: "test-app-v1",
		},
		{
			name: "v1beta2 license",
			wrapper: licensewrapper.LicenseWrapper{
				V2: &kotsv1beta2.License{
					Spec: kotsv1beta2.LicenseSpec{
						AppSlug: "test-app-v2",
					},
				},
			},
			expectIsV1:    false,
			expectIsV2:    true,
			expectAppSlug: "test-app-v2",
		},
		{
			name:          "empty wrapper",
			wrapper:       licensewrapper.LicenseWrapper{},
			expectIsV1:    false,
			expectIsV2:    false,
			expectAppSlug: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectIsV1, tt.wrapper.IsV1(), "IsV1() mismatch")
			assert.Equal(t, tt.expectIsV2, tt.wrapper.IsV2(), "IsV2() mismatch")
			assert.Equal(t, tt.expectAppSlug, tt.wrapper.GetAppSlug(), "GetAppSlug() mismatch")
		})
	}
}

// TestLicenseWrapper_V1Beta2_LoadFromBytes tests that v1beta2 licenses can be loaded from YAML
func TestLicenseWrapper_V1Beta2_LoadFromBytes(t *testing.T) {
	v1beta2YAML := []byte(`apiVersion: kots.io/v1beta2
kind: License
metadata:
  name: test-license-v2
spec:
  appSlug: test-app
  licenseID: test-license-id-v2
  licenseType: dev
  channelID: test-channel
  channelName: Test Channel
`)

	wrapper, err := licensewrapper.LoadLicenseFromBytes(v1beta2YAML)
	require.NoError(t, err, "should load v1beta2 license without error")

	assert.True(t, wrapper.IsV2(), "should be identified as v1beta2")
	assert.False(t, wrapper.IsV1(), "should not be identified as v1beta1")
	assert.Equal(t, "test-app", wrapper.GetAppSlug())
	assert.Equal(t, "test-license-id-v2", wrapper.GetLicenseID())
	assert.Equal(t, "test-channel", wrapper.GetChannelID())
}

// TestLicenseWrapper_V1Beta1_LoadFromBytes tests that v1beta1 licenses still work
func TestLicenseWrapper_V1Beta1_LoadFromBytes(t *testing.T) {
	v1beta1YAML := []byte(`apiVersion: kots.io/v1beta1
kind: License
metadata:
  name: test-license-v1
spec:
  appSlug: test-app
  licenseID: test-license-id-v1
  licenseType: dev
  channelID: test-channel
  channelName: Test Channel
`)

	wrapper, err := licensewrapper.LoadLicenseFromBytes(v1beta1YAML)
	require.NoError(t, err, "should load v1beta1 license without error")

	assert.True(t, wrapper.IsV1(), "should be identified as v1beta1")
	assert.False(t, wrapper.IsV2(), "should not be identified as v1beta2")
	assert.Equal(t, "test-app", wrapper.GetAppSlug())
	assert.Equal(t, "test-license-id-v1", wrapper.GetLicenseID())
	assert.Equal(t, "test-channel", wrapper.GetChannelID())
}

// TestLicenseWrapper_GetterMethods tests that getter methods work for both versions
func TestLicenseWrapper_GetterMethods(t *testing.T) {
	tests := []struct {
		name           string
		wrapper        licensewrapper.LicenseWrapper
		expectedValues map[string]interface{}
	}{
		{
			name: "v1beta1 with all fields",
			wrapper: licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "License",
					},
					Spec: kotsv1beta1.LicenseSpec{
						AppSlug:          "app-v1",
						LicenseID:        "lic-v1",
						ChannelID:        "chan-v1",
						ChannelName:      "Channel V1",
						IsSemverRequired: true,
					},
				},
			},
			expectedValues: map[string]interface{}{
				"AppSlug":          "app-v1",
				"LicenseID":        "lic-v1",
				"ChannelID":        "chan-v1",
				"IsSemverRequired": true,
			},
		},
		{
			name: "v1beta2 with all fields",
			wrapper: licensewrapper.LicenseWrapper{
				V2: &kotsv1beta2.License{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta2",
						Kind:       "License",
					},
					Spec: kotsv1beta2.LicenseSpec{
						AppSlug:          "app-v2",
						LicenseID:        "lic-v2",
						ChannelID:        "chan-v2",
						ChannelName:      "Channel V2",
						IsSemverRequired: true,
					},
				},
			},
			expectedValues: map[string]interface{}{
				"AppSlug":          "app-v2",
				"LicenseID":        "lic-v2",
				"ChannelID":        "chan-v2",
				"IsSemverRequired": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedValues["AppSlug"], tt.wrapper.GetAppSlug())
			assert.Equal(t, tt.expectedValues["LicenseID"], tt.wrapper.GetLicenseID())
			assert.Equal(t, tt.expectedValues["ChannelID"], tt.wrapper.GetChannelID())
			assert.Equal(t, tt.expectedValues["IsSemverRequired"], tt.wrapper.IsSemverRequired())
		})
	}
}

// TestLicenseWrapper_EmptyWrapper tests behavior with empty wrapper
func TestLicenseWrapper_EmptyWrapper(t *testing.T) {
	wrapper := licensewrapper.LicenseWrapper{}

	assert.False(t, wrapper.IsV1(), "empty wrapper should not be V1")
	assert.False(t, wrapper.IsV2(), "empty wrapper should not be V2")
	assert.Empty(t, wrapper.GetAppSlug(), "empty wrapper should return empty app slug")
	assert.Empty(t, wrapper.GetLicenseID(), "empty wrapper should return empty license ID")
	assert.Empty(t, wrapper.GetChannelID(), "empty wrapper should return empty channel ID")
}
