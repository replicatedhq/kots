package template

import (
	"encoding/json"
	"os"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
	"github.com/stretchr/testify/require"
)

func TestSharedLicenseFieldsAndEntitlements(t *testing.T) {
	data, err := os.ReadFile("testdata/license-fields.json")
	require.NoError(t, err)
	var fixture struct {
		License  json.RawMessage
		Expected map[string]string
	}
	require.NoError(t, json.Unmarshal(data, &fixture))
	var lic kotsv1beta1.License
	require.NoError(t, json.Unmarshal(fixture.License, &lic))
	ctx := licenseCtx{License: &licensewrapper.LicenseWrapper{V1: &lic}}

	for name, expected := range fixture.Expected {
		require.Equal(t, expected, ctx.licenseFieldValue(name), name)
		require.Empty(t, (licenseCtx{}).licenseFieldValue(name), name)
	}
}
