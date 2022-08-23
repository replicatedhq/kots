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
			name:      "MakeProxiedImageURL with multi part image parameter with @ character returns valid proxied image URL",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "image@image",
			want:      "host/proxy/slug/image",
		},
		{
			name:      "MakeProxiedImageURL multi part image parameter with : character returns valid proxied image URL",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "image:image",
			want:      "host/proxy/slug/image",
		},
		{
			name:      "MakeProxiedImageURL multi part image parameter with a namespace returns valid proxied image URL",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "namespace/image:image",
			want:      "host/proxy/slug/namespace/image",
		},
		{
			name:      "MakeProxiedImageURL multi part image parameter with : and @ characters returns valid proxied image URL",
			proxyHost: "host",
			appSlug:   "slug",
			image:     "image:tag@digest",
			want:      "host/proxy/slug/image",
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
