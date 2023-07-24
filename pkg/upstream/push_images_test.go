package upstream

import (
	"reflect"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

func Test_makeInstallationImages(t *testing.T) {
	tests := []struct {
		name   string
		images []kustomizetypes.Image
		want   []kotsv1beta1.InstallationImage
	}{
		{
			name: "preserves references",
			images: []kustomizetypes.Image{
				{
					Name:   "registry.replicated.com/appslug/taggedimage",
					NewTag: "v1.0.0",
				},
				{
					Name:   "registry.replicated.com/appslug/digestimage",
					Digest: "sha256:25dedae0aceb6b4fe5837a0acbacc6580453717f126a095aa05a3c6fcea14dd4",
				},
				{
					Name:   "registry.replicated.com/appslug/taganddigestimage",
					NewTag: "v1.0.0",
					Digest: "sha256:25dedae0aceb6b4fe5837a0acbacc6580453717f126a095aa05a3c6fcea14dd4",
				},
			},
			want: []kotsv1beta1.InstallationImage{
				{
					Image:     "registry.replicated.com/appslug/taggedimage:v1.0.0",
					IsPrivate: true,
				},
				{
					Image:     "registry.replicated.com/appslug/digestimage@sha256:25dedae0aceb6b4fe5837a0acbacc6580453717f126a095aa05a3c6fcea14dd4",
					IsPrivate: true,
				},
				{
					Image:     "registry.replicated.com/appslug/taganddigestimage@sha256:25dedae0aceb6b4fe5837a0acbacc6580453717f126a095aa05a3c6fcea14dd4",
					IsPrivate: true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := makeInstallationImages(tt.images); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makeInstallationImages() = %v, want %v", got, tt.want)
			}
		})
	}
}
