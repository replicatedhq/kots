package template

import (
	"testing"

	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/stretchr/testify/require"
)

func TestReplicatedImageFunctions(t *testing.T) {
	tests := []struct {
		name, expression, want string
		local                  registrytypes.RegistrySettings
		slug                   string
		wantError              string
	}{
		{name: "direct registry image", expression: `ReplicatedImageName "registry.custom/app/image:1"`, want: "registry.custom/app/image:1"},
		{name: "upstream registry image", expression: `ReplicatedImageName "registry.replicated.com/app/image:1"`, want: "registry.custom/app/image:1"},
		{name: "direct registry prefix", expression: `ReplicatedImageRegistry "registry.custom"`, want: "registry.custom"},
		{name: "upstream registry prefix", expression: `ReplicatedImageRegistry "registry.replicated.com"`, want: "registry.custom"},
		{name: "proxy image", expression: `ReplicatedImageName "nginx:1"`, slug: "app", want: "proxy.custom/proxy/app/docker.io/library/nginx:1"},
		{name: "public image", expression: `ReplicatedImageName "nginx:1" true`, slug: "app", want: "nginx:1"},
		{name: "custom proxy", expression: `ReplicatedImageName "proxy.replicated.com/proxy/app/docker.io/library/nginx:1"`, slug: "app", want: "proxy.custom/proxy/app/docker.io/library/nginx:1"},
		{name: "already proxied", expression: `ReplicatedImageName "proxy.custom/proxy/app/docker.io/library/nginx:1"`, slug: "app", want: "proxy.custom/proxy/app/docker.io/library/nginx:1"},
		{name: "proxy registry", expression: `ReplicatedImageRegistry "ghcr.io"`, slug: "app", want: "proxy.custom/proxy/app/ghcr.io"},
		{name: "public registry", expression: `ReplicatedImageRegistry "ghcr.io" true`, slug: "app", want: "ghcr.io"},
		{name: "nested local namespace", expression: `ReplicatedImageName "ghcr.io/org/image:1"`, local: registrytypes.RegistrySettings{Hostname: "registry.corp:5000", Namespace: "team/app"}, want: "registry.corp:5000/team/app/image:1"},
		{name: "local ignores noProxy", expression: `ReplicatedImageName "ghcr.io/org/image:1" true`, local: registrytypes.RegistrySettings{Hostname: "registry.corp:5000", Namespace: "team/app"}, want: "registry.corp:5000/team/app/image:1"},
		{name: "local registry prefix", expression: `ReplicatedImageRegistry "ghcr.io"`, local: registrytypes.RegistrySettings{Hostname: "registry.corp:5000", Namespace: "team/app"}, want: "registry.corp:5000/team/app"},
		{name: "empty online registry", expression: `ReplicatedImageRegistry ""`, want: ""},
		{name: "empty online registry bypass", expression: `ReplicatedImageRegistry "" true`, want: ""},
		{name: "empty local registry argument", expression: `ReplicatedImageRegistry ""`, local: registrytypes.RegistrySettings{Hostname: "registry.corp:5000", Namespace: "team/app"}, want: "registry.corp:5000/team/app"},
		{name: "local registry ignores unused malformed prefix", expression: `ReplicatedImageRegistry "INVALID:::registry"`, local: registrytypes.RegistrySettings{Hostname: "registry.corp:5000", Namespace: "team/app"}, want: "registry.corp:5000/team/app"},
		{name: "empty image", expression: `ReplicatedImageName ""`, want: ""},
		{name: "malformed registry", expression: `ReplicatedImageRegistry "INVALID:::registry"`, wantError: "parse registry reference"},
		{name: "malformed registry bypass", expression: `ReplicatedImageRegistry "INVALID:::registry" true`, wantError: "parse registry reference"},
		{name: "registry with port", expression: `ReplicatedImageRegistry "registry.example.com:5000"`, slug: "app", want: "proxy.custom/proxy/app/registry.example.com:5000"},
		{name: "registry proxy prefix", expression: `ReplicatedImageRegistry "proxy.custom/proxy/app/docker.io"`, slug: "app", want: "proxy.custom/proxy/app/docker.io"},
		{name: "malformed image", expression: `ReplicatedImageName "INVALID:::image"`, wantError: "parse image reference"},
		{name: "missing registry slug", expression: `ReplicatedImageRegistry "ghcr.io"`, wantError: `resolve proxy routing for registry "ghcr.io": application slug is required`},
		{name: "missing slug", expression: `ReplicatedImageName "nginx:1"`, wantError: `resolve proxy routing for image "nginx:1": application slug is required`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, _, err := NewBuilder(BuilderOptions{ApplicationInfo: &ApplicationInfo{Slug: tt.slug}, LocalRegistry: tt.local, VersionInfo: &VersionInfo{ReplicatedProxyDomain: "proxy.custom", ReplicatedRegistryDomain: "registry.custom"}})
			require.NoError(t, err)
			got, err := b.String("repl{{ " + tt.expression + " }}")
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			if tt.local.Hostname != "" && tt.expression == `ReplicatedImageName "ghcr.io/org/image:1"` {
				original, err := b.String(`repl{{ LocalImageName "ghcr.io/org/image:1" }}`)
				require.NoError(t, err)
				require.Equal(t, original, got)
			}
		})
	}
}
