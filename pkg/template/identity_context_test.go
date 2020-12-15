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
						ID: "KOTS Test Admin",
						RoleIDs: []string{
							"cluster-admin",
							"read-only",
						},
					},
					{
						ID: "KOTS Test Support",
						RoleIDs: []string{
							"support",
						},
					},
				},
				IdentityServiceAddress: "https://dex.kotsadmdevenv.com",
				ClientID:               "client-id",
				ClientSecret:           "client-secret",
			},
		},
		appInfo: &ApplicationInfo{
			Slug: "my-app",
		},
	}

	// an unpopulated identityCtx - should not error/panic
	nilCtx := identityCtx{}

	req.Equal(true, ctx.identityServiceEnabled())
	req.Equal(false, nilCtx.identityServiceEnabled())

	req.Equal("https://dex.kotsadmdevenv.com", ctx.identityServiceIssuerURL())
	req.Equal("", nilCtx.identityServiceIssuerURL())

	req.Equal("client-id", ctx.identityServiceClientID())
	req.Equal("", nilCtx.identityServiceClientID())

	req.Equal("client-secret", ctx.identityServiceClientSecret())
	req.Equal("", nilCtx.identityServiceClientSecret())

	req.Equal(map[string][]string{
		"KOTS Test Admin":   {"cluster-admin", "read-only"},
		"KOTS Test Support": {"support"},
	}, ctx.identityServiceRoles())
	req.Equal(map[string][]string{}, nilCtx.identityServiceRoles())

	req.Equal("my-app-dex", ctx.identityServiceName())
	req.Equal("", nilCtx.identityServiceName())

	req.Equal("5556", ctx.identityServicePort())
	req.Equal("", nilCtx.identityServicePort())
}
