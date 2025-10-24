package kotsutil_test

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsv1beta2 "github.com/replicatedhq/kotskinds/apis/kots/v1beta2"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Test 1: Load v1beta2 license from bytes
func TestLoadV1Beta2LicenseFromBytes(t *testing.T) {
	licenseYAML := `
apiVersion: kots.io/v1beta2
kind: License
metadata:
  name: test-license
spec:
  signature: c2lnbmF0dXJl
  appSlug: myapp
  licenseID: test-123
  customerName: Test Customer
  licenseType: dev
  entitlements:
    seats:
      title: Number of Seats
      value: 100
      valueType: Integer
      signature:
        v2: ZW50aXRsZW1lbnQtc2ln
`

	req := require.New(t)

	// Should load successfully
	license, err := kotsutil.LoadV1Beta2LicenseFromBytes([]byte(licenseYAML))
	req.NoError(err)
	req.NotNil(license)

	// Verify it's v1beta2
	req.Equal("kots.io/v1beta2", license.APIVersion)
	req.Equal("License", license.Kind)

	// Verify signature field populated (SHA-256 signature in v1beta2)
	req.NotNil(license.Spec.Signature)
	req.NotEmpty(license.Spec.Signature)

	// Verify basic fields
	req.Equal("test-123", license.Spec.LicenseID)
	req.Equal("myapp", license.Spec.AppSlug)
	req.Equal("Test Customer", license.Spec.CustomerName)

	// Verify entitlement V2 signature
	req.Contains(license.Spec.Entitlements, "seats")
	req.NotNil(license.Spec.Entitlements["seats"].Signature.V2)
	req.NotEmpty(license.Spec.Entitlements["seats"].Signature.V2)
}

// Test 2: Reject wrong version
func TestLoadV1Beta2LicenseFromBytes_RejectsV1Beta1(t *testing.T) {
	licenseYAML := `
apiVersion: kots.io/v1beta1
kind: License
metadata:
  name: test-license
spec:
  signature: c2lnbmF0dXJl
  appSlug: myapp
  licenseID: test-123
`

	req := require.New(t)

	// Should return error for v1beta1
	_, err := kotsutil.LoadV1Beta2LicenseFromBytes([]byte(licenseYAML))
	req.Error(err)
	req.Contains(err.Error(), "unexpected GVK")
}

// Test 3: Reject wrong kind
func TestLoadV1Beta2LicenseFromBytes_RejectsWrongKind(t *testing.T) {
	configYAML := `
apiVersion: kots.io/v1beta2
kind: Config
metadata:
  name: test-config
`

	req := require.New(t)

	_, err := kotsutil.LoadV1Beta2LicenseFromBytes([]byte(configYAML))
	req.Error(err)
	// Should fail to decode because Config is not a valid kind for v1beta2
	req.Contains(err.Error(), "failed to decode")
}

// Test 4: KotsKinds populates V1Beta2License
func TestAddKotsKinds_V1Beta2License(t *testing.T) {
	licenseYAML := `
apiVersion: kots.io/v1beta2
kind: License
metadata:
  name: test-license
spec:
  signature: c2lnbmF0dXJl
  appSlug: myapp
  licenseID: test-123
  customerName: Test Customer
`

	req := require.New(t)

	licenseData := []byte(licenseYAML)

	// Use KotsKindsFromMap which internally calls addKotsKinds
	kotsKindsMap := map[string][]byte{
		"license.yaml": licenseData,
	}
	result, err := kotsutil.KotsKindsFromMap(kotsKindsMap)
	req.NoError(err)

	// V1Beta2License should be populated
	req.NotNil(result.V1Beta2License)
	req.Equal("test-123", result.V1Beta2License.Spec.LicenseID)
	req.Equal("myapp", result.V1Beta2License.Spec.AppSlug)

	// License (v1beta1) should be nil
	req.Nil(result.License)
}

// Test 5: KotsKinds doesn't confuse v1beta1 and v1beta2
func TestAddKotsKinds_BothVersionsSeparate(t *testing.T) {
	multiDocYAML := `---
apiVersion: kots.io/v1beta1
kind: License
metadata:
  name: v1beta1-license
spec:
  signature: djFiZXRhMQ==
  appSlug: app1
  licenseID: license-v1
---
apiVersion: kots.io/v1beta2
kind: License
metadata:
  name: v1beta2-license
spec:
  signature: djFiZXRhMg==
  appSlug: app2
  licenseID: license-v2
`

	req := require.New(t)

	kotsKindsMap := map[string][]byte{
		"licenses.yaml": []byte(multiDocYAML),
	}
	kotsKinds, err := kotsutil.KotsKindsFromMap(kotsKindsMap)
	req.NoError(err)

	// Both should be populated
	req.NotNil(kotsKinds.License)
	req.NotNil(kotsKinds.V1Beta2License)

	// Each should have correct data
	req.Equal("license-v1", kotsKinds.License.Spec.LicenseID)
	req.Equal("license-v2", kotsKinds.V1Beta2License.Spec.LicenseID)
}

// Test 6: Marshal v1beta2 license
func TestMarshal_V1Beta2License(t *testing.T) {
	req := require.New(t)

	license := &kotsv1beta2.License{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta2",
			Kind:       "License",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-license",
		},
		Spec: kotsv1beta2.LicenseSpec{
			Signature:    []byte("signature"),
			AppSlug:      "myapp",
			LicenseID:    "test-123",
			CustomerName: "Test Customer",
		},
	}

	kotsKinds := kotsutil.KotsKinds{
		KotsApplication: kotsv1beta1.Application{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "kots.io/v1beta1",
				Kind:       "Application",
			},
		},
		Installation: kotsv1beta1.Installation{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "kots.io/v1beta1",
				Kind:       "Installation",
			},
		},
		V1Beta2License: license,
	}

	// Should marshal successfully
	result, err := kotsKinds.Marshal("kots.io", "v1beta2", "License")
	req.NoError(err)
	req.NotEmpty(result)

	// Verify YAML content
	req.Contains(result, "apiVersion: kots.io/v1beta2")
	req.Contains(result, "kind: License")
	req.Contains(result, "signature")
	req.Contains(result, "licenseID: test-123")
}

// Test 7: Marshal returns empty for nil v1beta2 license
func TestMarshal_V1Beta2License_Nil(t *testing.T) {
	req := require.New(t)

	kotsKinds := kotsutil.KotsKinds{
		KotsApplication: kotsv1beta1.Application{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "kots.io/v1beta1",
				Kind:       "Application",
			},
		},
		Installation: kotsv1beta1.Installation{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "kots.io/v1beta1",
				Kind:       "Installation",
			},
		},
		V1Beta2License: nil,
	}

	// Should return empty string
	result, err := kotsKinds.Marshal("kots.io", "v1beta2", "License")
	req.NoError(err)
	req.Empty(result)
}

// Test 8: Marshal v1beta1 still works (no regression)
func TestMarshal_V1Beta1License_NoRegression(t *testing.T) {
	req := require.New(t)

	license := &kotsv1beta1.License{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "License",
		},
		Spec: kotsv1beta1.LicenseSpec{
			Signature: []byte("signature"),
			AppSlug:   "myapp",
			LicenseID: "test-123",
		},
	}

	kotsKinds := kotsutil.KotsKinds{
		KotsApplication: kotsv1beta1.Application{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "kots.io/v1beta1",
				Kind:       "Application",
			},
		},
		Installation: kotsv1beta1.Installation{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "kots.io/v1beta1",
				Kind:       "Installation",
			},
		},
		License: license,
	}

	// Should still work
	result, err := kotsKinds.Marshal("kots.io", "v1beta1", "License")
	req.NoError(err)
	req.Contains(result, "apiVersion: kots.io/v1beta1")
	req.Contains(result, "signature")
}
