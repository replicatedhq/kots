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
	builder, _, err := NewBuilder(BuilderOptions{License: &licensewrapper.LicenseWrapper{V1: &lic}})
	require.NoError(t, err)
	empty, _, err := NewBuilder(BuilderOptions{})
	require.NoError(t, err)

	for name, expected := range fixture.Expected {
		got, err := builder.String(`repl{{ LicenseFieldValue "` + name + `" }}`)
		require.NoError(t, err)
		require.Equal(t, expected, got, name)
		got, err = empty.String(`repl{{ LicenseFieldValue "` + name + `" }}`)
		require.NoError(t, err)
		require.Empty(t, got, name)
	}
}
