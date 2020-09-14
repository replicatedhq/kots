package upstream

import (
	"testing"

	"github.com/stretchr/testify/require"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

func Test_buildImageAltNames(t *testing.T) {
	tests := []struct {
		name           string
		rewrittenImage kustomizetypes.Image
		want           []kustomizetypes.Image
	}{
		{
			name: "no rewriting",
			rewrittenImage: kustomizetypes.Image{
				Name:    "myregistry.com/repo/image:tag",
				NewName: "unchanged",
				NewTag:  "unchanged",
				Digest:  "unchanged",
			},
			want: []kustomizetypes.Image{
				{
					Name:    "myregistry.com/repo/image:tag",
					NewName: "unchanged",
					NewTag:  "unchanged",
					Digest:  "unchanged",
				},
			},
		},
		{
			name: "library latest",
			rewrittenImage: kustomizetypes.Image{
				Name:    "docker.io/library/image:latest",
				NewName: "unchanged",
				NewTag:  "unchanged",
				Digest:  "unchanged",
			},
			want: []kustomizetypes.Image{
				{
					Name:    "docker.io/library/image:latest",
					NewName: "unchanged",
					NewTag:  "unchanged",
					Digest:  "unchanged",
				},
				{
					Name:    "image:latest",
					NewName: "unchanged",
					NewTag:  "unchanged",
					Digest:  "unchanged",
				},
				{
					Name:    "image",
					NewName: "unchanged",
					NewTag:  "unchanged",
					Digest:  "unchanged",
				},
			},
		},
		{
			name: "docker.io, not library or latest",
			rewrittenImage: kustomizetypes.Image{
				Name:    "docker.io/myrepo/image:tag",
				NewName: "unchanged",
				NewTag:  "unchanged",
				Digest:  "unchanged",
			},
			want: []kustomizetypes.Image{
				{
					Name:    "docker.io/myrepo/image:tag",
					NewName: "unchanged",
					NewTag:  "unchanged",
					Digest:  "unchanged",
				},
				{
					Name:    "myrepo/image:tag",
					NewName: "unchanged",
					NewTag:  "unchanged",
					Digest:  "unchanged",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			got := buildImageAltNames(tt.rewrittenImage)

			req.Equal(tt.want, got)
		})
	}
}
