package image

import (
	"testing"

	dockerregistrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_IsPrivateImages(t *testing.T) {
	type args struct {
		baseImages      []string
		kotsKindsImages []string
		kotsKinds       *kotsutil.KotsKinds
	}

	tests := []struct {
		image string
		want  bool
	}{
		{
			image: "registry.replicated.com/appslug/image:version",
			want:  true,
		},
		{
			image: "quay.io/replicatedcom/qa-kots-2:alpine-3.4",
			want:  true,
		},
		{
			image: "quay.io/replicatedcom/qa-kots-1:alpine-3.5",
			want:  true,
		},
		{
			image: "quay.io/replicatedcom/qa-kots-3:alpine-3.6",
			want:  true,
		},
		{
			image: "quay.io/replicatedcom/someimage:1@sha256:25dedae0aceb6b4fe5837a0acbacc6580453717f126a095aa05a3c6fcea14dd4",
			want:  true,
		},
		{
			image: "testing.registry.com:5000/testing-ns/random-image:2",
			want:  true,
		},
		{
			image: "testing.registry.com:5000/testing-ns/random-image:1",
			want:  true,
		},
		{
			image: "redis:7@sha256:e96c03a6dda7d0f28e2de632048a3d34bb1636d0858b65ef9a554441c70f6633",
			want:  false,
		},
		{
			image: "nginx:1",
			want:  false,
		},
		{
			image: "busybox",
			want:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.image, func(t *testing.T) {
			req := require.New(t)

			got, err := IsPrivateImage(test.image, dockerregistrytypes.RegistryOptions{})
			req.NoError(err)

			assert.Equal(t, test.want, got)
		})
	}
}
