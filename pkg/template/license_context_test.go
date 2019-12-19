package template

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/assert"
)

func TestLicenseContext_dockercfg(t *testing.T) {
	ctx := LicenseCtx{
		License: &kotsv1beta1.License{
			Spec: kotsv1beta1.LicenseSpec{
				LicenseID: "abcdef",
			},
		},
	}

	expect := "eyJhdXRocyI6eyJwcm94eS5yZXBsaWNhdGVkLmNvbSI6eyJhdXRoIjoiWVdKalpHVm1PbUZpWTJSbFpnPT0ifSwicmVnaXN0cnkucmVwbGljYXRlZC5jb20iOnsiYXV0aCI6IllXSmpaR1ZtT21GaVkyUmxaZz09In19fQ=="
	dockercfg := ctx.licenseDockercfg()
	assert.Equal(t, dockercfg, expect)
}
