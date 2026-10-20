package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplicatedImageName(t *testing.T) {
	tests := []struct {
		name     string
		ctx      imageCtx
		image    string
		noProxy  []bool
		expected string
	}{
		{
			name: "online proxied",
			ctx: imageCtx{
				appSlug: "my-app",
				registry: RegistryInfo{
					ProxyDomain: "proxy.replicated.com",
				},
			},
			image:    "nginx:latest",
			expected: "proxy.replicated.com/proxy/my-app/docker.io/library/nginx:latest",
		},
		{
			name: "online noProxy",
			ctx: imageCtx{
				appSlug: "my-app",
				registry: RegistryInfo{
					ProxyDomain: "proxy.replicated.com",
				},
			},
			image:    "nginx:latest",
			noProxy:  []bool{true},
			expected: "nginx:latest",
		},
		{
			name: "airgap bare docker hub image stays as-is",
			ctx: imageCtx{
				mode:    RegistryModeAirgap,
				appSlug: "my-app", registry: RegistryInfo{RegistryAddress: "10.244.128.11:5000"},
			},
			image:    "nginx:latest",
			expected: "10.244.128.11:5000/nginx:latest",
		},
		{
			name: "airgap full reference uses flattened upload path",
			ctx: imageCtx{
				mode:     RegistryModeAirgap,
				registry: RegistryInfo{RegistryAddress: "10.244.128.11:5000"},
			},
			image:    "ghcr.io/myorg/myimage:v1",
			expected: "10.244.128.11:5000/myimage:v1",
		},
		{
			name: "airgap digest reference",
			ctx: imageCtx{
				mode:     RegistryModeAirgap,
				registry: RegistryInfo{RegistryAddress: "10.244.128.11:5000"},
			},
			image:    "ghcr.io/myorg/myimage@sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			expected: "10.244.128.11:5000/myimage@sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		},
		{
			name: "airgap tag+digest reference uses digest",
			ctx: imageCtx{
				mode:     RegistryModeAirgap,
				registry: RegistryInfo{RegistryAddress: "10.244.128.11:5000"},
			},
			image:    "registry.k8s.io/ingress-nginx/controller:v1.11.3@sha256:d56f135b6462cfc476447cfe564b83a45e8bb7da2774963b00d12161112270b7",
			expected: "10.244.128.11:5000/controller@sha256:d56f135b6462cfc476447cfe564b83a45e8bb7da2774963b00d12161112270b7",
		},
		{
			name: "airgap curlimages/curl uses flattened upload path",
			ctx: imageCtx{
				mode:     RegistryModeAirgap,
				registry: RegistryInfo{RegistryAddress: "10.244.128.11:5000"},
			},
			image:    "curlimages/curl:latest",
			expected: "10.244.128.11:5000/curl:latest",
		},
		{
			name: "online custom proxy domain",
			ctx: imageCtx{
				appSlug: "my-app",
				registry: RegistryInfo{
					ProxyDomain: "custom-proxy.example.com",
				},
			},
			image:    "redis:7",
			expected: "custom-proxy.example.com/proxy/my-app/docker.io/library/redis:7",
		},
		{
			name: "online already proxied with configured domain returns unchanged",
			ctx: imageCtx{
				appSlug:  "my-app",
				registry: RegistryInfo{ProxyDomain: "custom-proxy.example.com"},
			},
			image:    "custom-proxy.example.com/proxy/my-app/docker.io/library/nginx:latest",
			expected: "custom-proxy.example.com/proxy/my-app/docker.io/library/nginx:latest",
		},
		{
			name: "online proxy.replicated.com gets domain-swapped when custom domain configured",
			ctx: imageCtx{
				appSlug:  "my-app",
				registry: RegistryInfo{ProxyDomain: "custom-proxy.example.com"},
			},
			image:    "proxy.replicated.com/proxy/my-app/docker.io/library/nginx:latest",
			expected: "custom-proxy.example.com/proxy/my-app/docker.io/library/nginx:latest",
		},
		{
			name:     "empty image",
			ctx:      imageCtx{},
			image:    "",
			expected: "",
		},
		{
			name: "BYO bare docker hub image stays as-is",
			ctx: imageCtx{
				mode:     RegistryModeBYO,
				registry: RegistryInfo{BYORegistryAddress: "registry.corp:5000/vendorflow"},
			},
			image:    "nginx:latest",
			expected: "registry.corp:5000/vendorflow/nginx:latest",
		},
		{
			name: "BYO custom proxy prefix without explicit upstream host preserves remaining path",
			ctx: imageCtx{
				mode:     RegistryModeBYO,
				registry: RegistryInfo{BYORegistryAddress: "registry.corp:5000/vendorflow", ProxyDomain: "images.vendorflowinc.com"},
			},
			image:    "images.vendorflowinc.com/proxy/vendorflow/amazon/aws-cli:latest",
			expected: "registry.corp:5000/vendorflow/aws-cli:latest",
		},
		{
			name: "BYO full upstream reference replaces only the host",
			ctx: imageCtx{
				mode:     RegistryModeBYO,
				registry: RegistryInfo{BYORegistryAddress: "registry.corp:5000/vendorflow"},
			},
			image:    "ghcr.io/myorg/myimage:v1",
			expected: "registry.corp:5000/vendorflow/myimage:v1",
		},
		{
			name: "BYO replicated proxy prefix is stripped through upstream host (explicit library path preserved)",
			ctx: imageCtx{
				mode:     RegistryModeBYO,
				registry: RegistryInfo{BYORegistryAddress: "registry.corp:5000/vendorflow"},
			},
			image:    "proxy.replicated.com/proxy/my-app/docker.io/library/nginx:latest",
			expected: "registry.corp:5000/vendorflow/nginx:latest",
		},
		{
			name: "BYO custom proxy domain prefix is stripped through upstream host",
			ctx: imageCtx{
				mode:     RegistryModeBYO,
				registry: RegistryInfo{BYORegistryAddress: "registry.corp:5000/vendorflow", ProxyDomain: "proxy.myapp.com"},
			},
			image:    "proxy.myapp.com/proxy/my-app/ghcr.io/myorg/myimage:v1",
			expected: "registry.corp:5000/vendorflow/myimage:v1",
		},
		{
			name: "BYO replicated.com host without proxy segment is treated as a regular ref",
			ctx: imageCtx{
				mode:     RegistryModeBYO,
				registry: RegistryInfo{BYORegistryAddress: "registry.corp:5000/vendorflow"},
			},
			image:    "proxy.replicated.com/library/calico-cni:v3.29.7",
			expected: "registry.corp:5000/vendorflow/calico-cni:v3.29.7",
		},
		{
			name: "BYO digest reference preserves repository path",
			ctx: imageCtx{
				mode:     RegistryModeBYO,
				registry: RegistryInfo{BYORegistryAddress: "registry.corp:5000/vendorflow"},
			},
			image:    "ghcr.io/myorg/myimage@sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			expected: "registry.corp:5000/vendorflow/myimage@sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		},
		{
			name: "BYO address without namespace path",
			ctx: imageCtx{
				mode:     RegistryModeBYO,
				registry: RegistryInfo{BYORegistryAddress: "registry.corp:5000"},
			},
			image:    "ghcr.io/myorg/myimage:v1",
			expected: "registry.corp:5000/myimage:v1",
		},
		{
			name: "BYO noProxy is ignored — image still rewritten",
			ctx: imageCtx{
				mode:     RegistryModeBYO,
				registry: RegistryInfo{BYORegistryAddress: "registry.corp:5000/vendorflow"},
			},
			image:    "ghcr.io/myorg/myimage:v1",
			noProxy:  []bool{true},
			expected: "registry.corp:5000/vendorflow/myimage:v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.ctx.replicatedImageName(tt.image, tt.noProxy...)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplicatedImageName_Error(t *testing.T) {
	t.Run("invalid image reference", func(t *testing.T) {
		ctx := imageCtx{
			mode:    RegistryModeAirgap,
			appSlug: "my-app", registry: RegistryInfo{RegistryAddress: "10.244.128.11:5000"},
		}
		_, err := ctx.replicatedImageName("INVALID:::image")
		assert.Error(t, err)
	})

	t.Run("airgap without registry address", func(t *testing.T) {
		ctx := imageCtx{
			mode:    RegistryModeAirgap,
			appSlug: "my-app",
		}
		_, err := ctx.replicatedImageName("nginx:latest")
		assert.Error(t, err)
	})

	t.Run("online missing proxy domain", func(t *testing.T) {
		ctx := imageCtx{
			appSlug: "my-app",
		}
		_, err := ctx.replicatedImageName("nginx:latest")
		assert.Error(t, err)
	})

	t.Run("online missing app slug", func(t *testing.T) {
		ctx := imageCtx{
			registry: RegistryInfo{ProxyDomain: "proxy.replicated.com"},
		}
		_, err := ctx.replicatedImageName("nginx:latest")
		assert.Error(t, err)
	})

	t.Run("BYO without registry address", func(t *testing.T) {
		ctx := imageCtx{mode: RegistryModeBYO}
		_, err := ctx.replicatedImageName("nginx:latest")
		assert.Error(t, err)
	})

	t.Run("BYO invalid image reference", func(t *testing.T) {
		ctx := imageCtx{
			mode:     RegistryModeBYO,
			registry: RegistryInfo{BYORegistryAddress: "registry.corp:5000/vendorflow"},
		}
		_, err := ctx.replicatedImageName("INVALID:::image")
		assert.Error(t, err)
	})
}

func TestReplicatedImageRegistry(t *testing.T) {
	tests := []struct {
		name     string
		ctx      imageCtx
		registry string
		noProxy  []bool
		expected string
	}{
		{
			name: "online returns proxy path with upstream registry",
			ctx: imageCtx{
				appSlug:  "my-app",
				registry: RegistryInfo{ProxyDomain: "proxy.replicated.com"},
			},
			registry: "docker.io",
			expected: "proxy.replicated.com/proxy/my-app/docker.io",
		},
		{
			name: "online noProxy returns original",
			ctx: imageCtx{
				appSlug:  "my-app",
				registry: RegistryInfo{ProxyDomain: "proxy.replicated.com"},
			},
			registry: "docker.io",
			noProxy:  []bool{true},
			expected: "docker.io",
		},
		{
			name: "airgap returns local registry",
			ctx: imageCtx{
				mode:     RegistryModeAirgap,
				registry: RegistryInfo{RegistryAddress: "10.244.128.11:5000"},
			},
			registry: "docker.io",
			expected: "10.244.128.11:5000",
		},
		{
			name: "online already proxied returns unchanged",
			ctx: imageCtx{
				appSlug:  "my-app",
				registry: RegistryInfo{ProxyDomain: "proxy.myapp.com"},
			},
			registry: "proxy.myapp.com/proxy/my-app/ghcr.io",
			expected: "proxy.myapp.com/proxy/my-app/ghcr.io",
		},
		{
			name: "online proxy domain only returns unchanged",
			ctx: imageCtx{
				appSlug:  "my-app",
				registry: RegistryInfo{ProxyDomain: "proxy.myapp.com"},
			},
			registry: "proxy.myapp.com",
			expected: "proxy.myapp.com",
		},
		{
			name: "online custom proxy domain with upstream",
			ctx: imageCtx{
				appSlug:  "my-app",
				registry: RegistryInfo{ProxyDomain: "custom-proxy.example.com"},
			},
			registry: "ghcr.io",
			expected: "custom-proxy.example.com/proxy/my-app/ghcr.io",
		},
		{
			name: "online registry.replicated.com gets proxied",
			ctx: imageCtx{
				appSlug:  "my-app",
				registry: RegistryInfo{ProxyDomain: "proxy.replicated.com"},
			},
			registry: "registry.replicated.com",
			expected: "proxy.replicated.com/proxy/my-app/registry.replicated.com",
		},
		{
			name: "online proxy.replicated.com gets domain-swapped when custom domain configured",
			ctx: imageCtx{
				appSlug:  "my-app",
				registry: RegistryInfo{ProxyDomain: "custom-proxy.example.com"},
			},
			registry: "proxy.replicated.com",
			expected: "custom-proxy.example.com",
		},
		{
			name: "online proxy.replicated.com with path gets domain-swapped",
			ctx: imageCtx{
				appSlug:  "my-app",
				registry: RegistryInfo{ProxyDomain: "custom-proxy.example.com"},
			},
			registry: "proxy.replicated.com/proxy/my-app/ghcr.io",
			expected: "custom-proxy.example.com/proxy/my-app/ghcr.io",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.ctx.replicatedImageRegistry(tt.registry, tt.noProxy...)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplicatedImageRegistry_Error(t *testing.T) {
	t.Run("airgap without registry address", func(t *testing.T) {
		ctx := imageCtx{mode: RegistryModeAirgap}
		_, err := ctx.replicatedImageRegistry("docker.io")
		assert.Error(t, err)
	})

	t.Run("online without proxy domain", func(t *testing.T) {
		ctx := imageCtx{appSlug: "my-app"}
		_, err := ctx.replicatedImageRegistry("docker.io")
		assert.Error(t, err)
	})

	t.Run("online without app slug", func(t *testing.T) {
		ctx := imageCtx{registry: RegistryInfo{ProxyDomain: "proxy.replicated.com"}}
		_, err := ctx.replicatedImageRegistry("docker.io")
		assert.Error(t, err)
	})
}

func TestReplicatedDomainFunctions(t *testing.T) {
	ctx := imageCtx{
		registry: RegistryInfo{
			ProxyDomain:    "custom-proxy.example.com",
			RegistryDomain: "custom-registry.example.com",
		},
	}

	assert.Equal(t, "custom-proxy.example.com", ctx.replicatedProxyDomain())
	assert.Equal(t, "custom-registry.example.com", ctx.replicatedRegistryDomain())

	// empty returns empty
	empty := imageCtx{}
	assert.Equal(t, "", empty.replicatedProxyDomain())
	assert.Equal(t, "", empty.replicatedRegistryDomain())
}
