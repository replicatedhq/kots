package kotsutil_test

import (
	"strings"
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsv1beta2 "github.com/replicatedhq/kotskinds/apis/kots/v1beta2"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestKotsKinds_GetLicenseVersion(t *testing.T) {
	tests := []struct {
		name     string
		license  *licensewrapper.LicenseWrapper
		expected string
	}{
		{
			name:     "v1beta1 license",
			license:  &licensewrapper.LicenseWrapper{V1: &kotsv1beta1.License{}},
			expected: "v1beta1",
		},
		{
			name:     "v1beta2 license",
			license:  &licensewrapper.LicenseWrapper{V2: &kotsv1beta2.License{}},
			expected: "v1beta2",
		},
		{
			name:     "no license",
			license:  &licensewrapper.LicenseWrapper{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &kotsutil.KotsKinds{License: tt.license}
			got := k.GetLicenseVersion()
			if got != tt.expected {
				t.Errorf("GetLicenseVersion() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestKotsKinds_HasLicense(t *testing.T) {
	tests := []struct {
		name     string
		license  *licensewrapper.LicenseWrapper
		expected bool
	}{
		{
			name:     "has v1beta1 license",
			license:  &licensewrapper.LicenseWrapper{V1: &kotsv1beta1.License{}},
			expected: true,
		},
		{
			name:     "has v1beta2 license",
			license:  &licensewrapper.LicenseWrapper{V2: &kotsv1beta2.License{}},
			expected: true,
		},
		{
			name:     "no license",
			license:  &licensewrapper.LicenseWrapper{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &kotsutil.KotsKinds{License: tt.license}
			got := k.HasLicense()
			if got != tt.expected {
				t.Errorf("HasLicense() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestKotsKinds_Marshal_V1Beta2License(t *testing.T) {
	// Test that Marshal correctly handles v1beta2 licenses
	k := kotsutil.KotsKinds{
		License: &licensewrapper.LicenseWrapper{
			V2: &kotsv1beta2.License{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kots.io/v1beta2",
					Kind:       "License",
				},
				Spec: kotsv1beta2.LicenseSpec{
					AppSlug: "my-app",
				},
			},
		},
	}

	yaml, err := k.Marshal("kots.io", "v1beta2", "License")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if !strings.Contains(yaml, "apiVersion: kots.io/v1beta2") {
		t.Error("expected v1beta2 in marshaled YAML")
	}
}

func TestKotsKinds_Marshal_V1Beta1License(t *testing.T) {
	// Test that Marshal correctly handles v1beta1 licenses
	k := kotsutil.KotsKinds{
		License: &licensewrapper.LicenseWrapper{
			V1: &kotsv1beta1.License{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kots.io/v1beta1",
					Kind:       "License",
				},
				Spec: kotsv1beta1.LicenseSpec{
					AppSlug: "my-app",
				},
			},
		},
	}

	// Test marshaling as v1beta1
	yaml, err := k.Marshal("kots.io", "v1beta1", "License")
	require.NoError(t, err)
	require.Contains(t, yaml, "apiVersion: kots.io/v1beta1")

	// Test that v1beta1 can be marshaled when v1beta2 is requested (backward compat)
	yaml, err = k.Marshal("kots.io", "v1beta2", "License")
	require.NoError(t, err)
	require.Contains(t, yaml, "apiVersion: kots.io/v1beta1")
}
