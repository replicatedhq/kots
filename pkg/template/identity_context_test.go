package template

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/crypto"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/require"
)

func TestIdentityContext(t *testing.T) {
	req := require.New(t)

	err := crypto.NewAESCipher()
	req.NoError(err)

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
				ClientSecret:           kotsv1beta1.NewStringValueOrEncrypted("client-secret"),
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

	req.Equal("client-id", ctx.identityServiceClientID())
	req.Equal("", nilCtx.identityServiceClientID())

	val, err := ctx.identityServiceClientSecret()
	req.NoError(err)
	req.Equal("client-secret", val)
	val, err = nilCtx.identityServiceClientSecret()
	req.NoError(err)
	req.Equal("", val)

	req.Equal(map[string]interface{}{
		"KOTS Test Admin":   []string{"cluster-admin", "read-only"},
		"KOTS Test Support": []string{"support"},
	}, ctx.identityServiceRoles())
	req.Equal(map[string]interface{}{}, nilCtx.identityServiceRoles())

	req.Equal("my-app-dex", ctx.identityServiceName())
	req.Equal("", nilCtx.identityServiceName())

	req.Equal("5556", ctx.identityServicePort())
	req.Equal("", nilCtx.identityServicePort())
}
