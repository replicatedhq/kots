package registry

import "testing"

func Test_MakeProxiedImageURL(t *testing.T) {
	tests := []struct {
		name      string
		proxyHost string
		appSlug   string
		image     string
		want      string
	}{
		{
			name:      "untagged image",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "image",
			want:      "host/proxy/slug/image",
		},
		{
			name:      "untagged image on non-ported registry",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "registry/image",
			want:      "host/proxy/slug/registry/image",
		},
		{
			name:      "untagged image on ported registry",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "registry:5000/image",
			want:      "host/proxy/slug/registry:5000/image",
		},
		{
			name:      "tagged image",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "image:tag",
			want:      "host/proxy/slug/image",
		},
		{
			name:      "tagged image on non-ported registry",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "registry/image:tag",
			want:      "host/proxy/slug/registry/image",
		},
		{
			name:      "untagged image on ported registry",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "registry:5000/image:tag",
			want:      "host/proxy/slug/registry:5000/image",
		},
		{
			name:      "digest image",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "image@digest",
			want:      "host/proxy/slug/image",
		},
		{
			name:      "digest image on non-ported registry",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "registry/image@digest",
			want:      "host/proxy/slug/registry/image",
		},
		{
			name:      "digest image on ported registry",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "registry:5000/image@digest",
			want:      "host/proxy/slug/registry:5000/image",
		},
		{
			name:      "tag and digest image",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "image:tag@digest",
			want:      "host/proxy/slug/image",
		},
		{
			name:      "tag and digest image on non-ported registry",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "registry/image:tag@digest",
			want:      "host/proxy/slug/registry/image",
		},
		{
			name:      "tag and digest image on ported registry",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "registry:5000/image:tag@digest",
			want:      "host/proxy/slug/registry:5000/image",
		},
		// ---- test cases for images that are already proxied ---- //
		{
			name:      "untagged proxied image",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "host/proxy/slug/image",
			want:      "host/proxy/slug/image",
		},
		{
			name:      "untagged proxied image on non-ported registry",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "host/proxy/slug/registry/image",
			want:      "host/proxy/slug/registry/image",
		},
		{
			name:      "untagged proxied image on ported registry",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "host/proxy/slug/registry:5000/image",
			want:      "host/proxy/slug/registry:5000/image",
		},
		{
			name:      "tagged proxied image",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "host/proxy/slug/image:tag",
			want:      "host/proxy/slug/image",
		},
		{
			name:      "tagged proxied image on non-ported registry",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "host/proxy/slug/registry/image:tag",
			want:      "host/proxy/slug/registry/image",
		},
		{
			name:      "untagged proxied image on ported registry",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "host/proxy/slug/registry:5000/image:tag",
			want:      "host/proxy/slug/registry:5000/image",
		},
		{
			name:      "digest proxied image",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "host/proxy/slug/image@digest",
			want:      "host/proxy/slug/image",
		},
		{
			name:      "digest proxied image on non-ported registry",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "host/proxy/slug/registry/image@digest",
			want:      "host/proxy/slug/registry/image",
		},
		{
			name:      "digest proxied image on ported registry",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "host/proxy/slug/registry:5000/image@digest",
			want:      "host/proxy/slug/registry:5000/image",
		},
		{
			name:      "tag and digest proxied image",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "host/proxy/slug/image:tag@digest",
			want:      "host/proxy/slug/image",
		},
		{
			name:      "tag and digest proxied image on non-ported registry",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "host/proxy/slug/registry/image:tag@digest",
			want:      "host/proxy/slug/registry/image",
		},
		{
			name:      "tag and digest proxied image on ported registry",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "host/proxy/slug/registry:5000/image:tag@digest",
			want:      "host/proxy/slug/registry:5000/image",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := MakeProxiedImageURL(tt.proxyHost, tt.appSlug, tt.image); res != tt.want {
				t.Errorf("MakeProxiedImageURL() = %v, want %v", res, tt.want)
			}
		})
	}
}
