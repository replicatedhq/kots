package template

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/require"
)

func TestIdentityContext(t *testing.T) {
	req := require.New(t)

	// a properly populated identityCtx - should return the appropriate values
	ctx := identityCtx{
		identityConfig: &kotsv1beta1.IdentityConfig{
			Spec: kotsv1beta1.IdentityConfigSpec{
				Enabled: true,
				Groups: []kotsv1beta1.IdentityConfigGroup{
					{
						ID: "cluster-admin",
					},
				},
				IdentityServiceAddress: "https://dex.kotsadmdevenv.com",
				ClientID:               "client-id",
				ClientSecret:           "client-secret",
			},
		},
	}

	// an unpopulated identityCtx - should not error/panic
	nilCtx := identityCtx{}

	req.Equal(true, ctx.identityServiceEnabled())
	req.Equal(false, nilCtx.identityServiceEnabled())

	// TODO: (salah)
	// req.Equal("", ctx.identityServiceIssuerURL())
	// req.Equal("", nilCtx.identityServiceIssuerURL())

	req.Equal("client-id", ctx.identityServiceClientID())
	req.Equal("", nilCtx.identityServiceClientID())

	req.Equal("client-secret", ctx.identityServiceClientSecret())
	req.Equal("", nilCtx.identityServiceClientSecret())

	req.Equal([]string{"cluster-admin"}, ctx.identityServiceRestrictedGroups())
	req.Equal([]string{}, nilCtx.identityServiceRestrictedGroups())

	// TODO: (salah)
	// req.Equal("", ctx.identityServiceRoles())
	// req.Equal("", nilCtx.identityServiceRoles())
}
