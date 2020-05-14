package v1beta1tests

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "go.undefinedlabs.com/scopeagent/autoinstrument"
	"k8s.io/client-go/kubernetes/scheme"
)

func Test_License(t *testing.T) {
	data := `apiVersion: kots.io/v1beta1
kind: License
metadata:
  name: local
spec:
  licenseID: abcdef
  appSlug: my-app
  endpoint: 'http://localhost:30016'
  entitlements:
    expires_at:
      title: Expiration
      description: License Expiration
      value: ""
    has-product-2:
      title: Has Product 2
      value: "test"
    is_vip:
      title: Is VIP
      value: false
    num_seats:
      title: Number Of Seats
      value: 10
    sdzf:
      title: sdf
      value: 1
    test:
      title: test
      value: "123asd"
  signature: IA==`

	kotsscheme.AddToScheme(scheme.Scheme)

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode([]byte(data), nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "kots.io", gvk.Group)
	assert.Equal(t, "v1beta1", gvk.Version)
	assert.Equal(t, "License", gvk.Kind)

	license := obj.(*kotsv1beta1.License)

	assert.Equal(t, "abcdef", license.Spec.LicenseID)

	entitlements := license.Spec.Entitlements
	assert.Len(t, entitlements, 6)

	expiresAt := entitlements["expires_at"]
	assert.NotNil(t, expiresAt)
	assert.Equal(t, "Expiration", expiresAt.Title)
	assert.Equal(t, "License Expiration", expiresAt.Description)

	numSeats := entitlements["num_seats"]
	assert.NotNil(t, numSeats)
	assert.Equal(t, "Number Of Seats", numSeats.Title)
	assert.Equal(t, "", numSeats.Description)

	testField := entitlements["test"]
	assert.NotNil(t, testField)
	assert.Equal(t, "test", testField.Title)
	assert.Equal(t, "123asd", testField.Value.Value())

}
