package image

import (
	"fmt"
	"testing"

	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

func Test_ImageNameFromNameParts(t *testing.T) {
	tests := []struct {
		name        string
		parts       []string
		registryOps registry.RegistryOptions
		expected    kustomizetypes.Image
		isError     bool
	}{
		{
			name:  "bad name format",
			parts: []string{"quay.io", "latest"},
			registryOps: registry.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somebigbank",
			},
			expected: kustomizetypes.Image{},
			isError:  true,
		},
		{
			name:  "ECR style image",
			parts: []string{"411111111111.dkr.ecr.us-west-1.amazonaws.com", "myrepo", "v0.0.1"},
			registryOps: registry.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somebigbank",
			},
			expected: kustomizetypes.Image{
				Name:    "411111111111.dkr.ecr.us-west-1.amazonaws.com/myrepo:v0.0.1",
				NewName: "localhost:5000/somebigbank/myrepo",
				NewTag:  "v0.0.1",
				Digest:  "",
			},
			isError: false,
		},
		{
			name:  "four parts with tag",
			parts: []string{"quay.io", "someorg", "debian", "0.1"},
			registryOps: registry.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somebigbank",
			},
			expected: kustomizetypes.Image{
				Name:    "quay.io/someorg/debian:0.1",
				NewName: "localhost:5000/somebigbank/debian",
				NewTag:  "0.1",
				Digest:  "",
			},
			isError: false,
		},
		{
			name:  "five parts with sha",
			parts: []string{"quay.io", "someorg", "debian", "sha256", "1234567890abcdef"},
			registryOps: registry.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somebigbank",
			},
			expected: kustomizetypes.Image{
				Name:    "quay.io/someorg/debian@sha256:1234567890abcdef",
				NewName: "localhost:5000/somebigbank/debian",
				NewTag:  "",
				Digest:  "1234567890abcdef",
			},
			isError: false,
		},
		{
			name:  "no namespace",
			parts: []string{"quay.io", "someorg", "debian", "0.1"},
			registryOps: registry.RegistryOptions{
				Endpoint: "localhost:5000",
			},
			expected: kustomizetypes.Image{
				Name:    "quay.io/someorg/debian:0.1",
				NewName: "localhost:5000/debian",
				NewTag:  "0.1",
				Digest:  "",
			},
			isError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			image, err := ImageInfoFromFile(test.registryOps, test.parts)
			if test.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, image)
			}
		})
	}
}

func Test_DestRef(t *testing.T) {
	registryOps := registry.RegistryOptions{
		Endpoint:  "localhost:5000",
		Namespace: "somebigbank",
	}

	type args struct {
		registry registry.RegistryOptions
		srcImage string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "ECR style image",
			args: args{
				registry: registryOps,
				srcImage: "411111111111.dkr.ecr.us-west-1.amazonaws.com/myrepo:v0.0.1",
			},
			want: fmt.Sprintf("%s/%s/myrepo:v0.0.1", registryOps.Endpoint, registryOps.Namespace),
		},
		{
			name: "Quay image with tag",
			args: args{
				registry: registryOps,
				srcImage: "quay.io/someorg/debian:0.1",
			},
			want: fmt.Sprintf("%s/%s/debian:0.1", registryOps.Endpoint, registryOps.Namespace),
		},
		{
			name: "Quay image with digest",
			args: args{
				registry: registryOps,
				srcImage: "quay.io/someorg/debian@sha256:mytestdigest",
			},
			want: fmt.Sprintf("%s/%s/debian@sha256:mytestdigest", registryOps.Endpoint, registryOps.Namespace),
		},
		{
			name: "No Namespace",
			args: args{
				registry: registry.RegistryOptions{
					Endpoint: "localhost:5000",
				},
				srcImage: "quay.io/someorg/debian:0.1",
			},
			want: fmt.Sprintf("%s/debian:0.1", registryOps.Endpoint),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DestRef(tt.args.registry, tt.args.srcImage); got != tt.want {
				t.Errorf("DestImageName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_BuildImageAltNames(t *testing.T) {
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
					Name:    "library/image:latest",
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
		{
			name: "no docker.io, not library or latest",
			rewrittenImage: kustomizetypes.Image{
				Name:    "myrepo/image:tag",
				NewName: "unchanged",
				NewTag:  "unchanged",
				Digest:  "unchanged",
			},
			want: []kustomizetypes.Image{
				{
					Name:    "myrepo/image:tag",
					NewName: "unchanged",
					NewTag:  "unchanged",
					Digest:  "unchanged",
				},
				{
					Name:    "docker.io/myrepo/image:tag",
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

			got := BuildImageAltNames(tt.rewrittenImage)

			req.Equal(tt.want, got)
		})
	}
}

func Test_kustomizeImage(t *testing.T) {
	tests := []struct {
		name         string
		destRegistry registry.RegistryOptions
		image        string
		want         []kustomizetypes.Image
	}{
		{
			name: "naked image",
			destRegistry: registry.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somebigbank",
			},
			image: "redis",
			want: []kustomizetypes.Image{
				{
					Name:    "redis",
					NewName: "localhost:5000/somebigbank/redis",
					NewTag:  "latest",
					Digest:  "",
				},
				{
					Name:    "docker.io/library/redis",
					NewName: "localhost:5000/somebigbank/redis",
					NewTag:  "latest",
					Digest:  "",
				},
				{
					Name:    "library/redis",
					NewName: "localhost:5000/somebigbank/redis",
					NewTag:  "latest",
					Digest:  "",
				},
			},
		},
		{
			name: "naked tagged image",
			destRegistry: registry.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somebigbank",
			},
			image: "redis:v1",
			want: []kustomizetypes.Image{
				{
					Name:    "redis",
					NewName: "localhost:5000/somebigbank/redis",
					NewTag:  "v1",
					Digest:  "",
				},
				{
					Name:    "docker.io/library/redis",
					NewName: "localhost:5000/somebigbank/redis",
					NewTag:  "v1",
					Digest:  "",
				},
				{
					Name:    "library/redis",
					NewName: "localhost:5000/somebigbank/redis",
					NewTag:  "v1",
					Digest:  "",
				},
			},
		},
		{
			name: "naked contentAddressableSha image",
			destRegistry: registry.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somesmallcorp",
			},
			image: "redis@sha256:mytestdigest",
			want: []kustomizetypes.Image{
				{
					Name:    "redis",
					NewName: "localhost:5000/somesmallcorp/redis",
					NewTag:  "",
					Digest:  "mytestdigest",
				},
				{
					Name:    "docker.io/library/redis",
					NewName: "localhost:5000/somesmallcorp/redis",
					NewTag:  "",
					Digest:  "mytestdigest",
				},
				{
					Name:    "library/redis",
					NewName: "localhost:5000/somesmallcorp/redis",
					NewTag:  "",
					Digest:  "mytestdigest",
				},
			},
		},
		{
			name: "tagged image",
			destRegistry: registry.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somebigbank",
			},
			image: "library/redis:v1",
			want: []kustomizetypes.Image{
				{
					Name:    "library/redis",
					NewName: "localhost:5000/somebigbank/redis",
					NewTag:  "v1",
					Digest:  "",
				},
				{
					Name:    "redis",
					NewName: "localhost:5000/somebigbank/redis",
					NewTag:  "v1",
					Digest:  "",
				},
				{
					Name:    "docker.io/library/redis",
					NewName: "localhost:5000/somebigbank/redis",
					NewTag:  "v1",
					Digest:  "",
				},
			},
		},
		{
			name: "quay.io tagged image",
			destRegistry: registry.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somebigbank",
			},
			image: "quay.io/library/redis:v1",
			want: []kustomizetypes.Image{
				{
					Name:    "quay.io/library/redis",
					NewName: "localhost:5000/somebigbank/redis",
					NewTag:  "v1",
					Digest:  "",
				},
			},
		},
		{
			name: "ported registry tagged image",
			destRegistry: registry.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somebigbank",
			},
			image: "example.com:5000/library/redis:v1",
			want: []kustomizetypes.Image{
				{
					Name:    "example.com:5000/library/redis",
					NewName: "localhost:5000/somebigbank/redis",
					NewTag:  "v1",
					Digest:  "",
				},
			},
		},
		{
			name: "ported registry untagged image",
			destRegistry: registry.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somebigbank",
			},
			image: "example.com:5000/library/redis",
			want: []kustomizetypes.Image{
				{
					Name:    "example.com:5000/library/redis",
					NewName: "localhost:5000/somebigbank/redis",
					NewTag:  "latest",
					Digest:  "",
				},
			},
		},
		{
			name: "fluent/fluentd:v1.7",
			destRegistry: registry.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somebigbank",
			},
			image: "fluent/fluentd:v1.7",
			want: []kustomizetypes.Image{
				{
					Name:    "fluent/fluentd",
					NewName: "localhost:5000/somebigbank/fluentd",
					NewTag:  "v1.7",
					Digest:  "",
				},
				{
					Name:    "docker.io/fluent/fluentd",
					NewName: "localhost:5000/somebigbank/fluentd",
					NewTag:  "v1.7",
					Digest:  "",
				},
			},
		},
		{
			name: "no namespace",
			destRegistry: registry.RegistryOptions{
				Endpoint: "localhost:5000",
			},
			image: "library/redis:v1",
			want: []kustomizetypes.Image{
				{
					Name:    "library/redis",
					NewName: "localhost:5000/redis",
					NewTag:  "v1",
					Digest:  "",
				},
				{
					Name:    "redis",
					NewName: "localhost:5000/redis",
					NewTag:  "v1",
					Digest:  "",
				},
				{
					Name:    "docker.io/library/redis",
					NewName: "localhost:5000/redis",
					NewTag:  "v1",
					Digest:  "",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got, err := kustomizeImage(tt.destRegistry, tt.image)
			req.NoError(err)
			req.Equal(tt.want, got)
		})
	}
}

func Test_stripImageTag(t *testing.T) {
	tests := []struct {
		name  string
		image string
		want  string
	}{
		{
			name:  "untagged image",
			image: "myimage",
			want:  "myimage",
		},
		{
			name:  "untagged image on ported registry",
			image: "example.com:5000/myimage",
			want:  "example.com:5000/myimage",
		},
		{
			name:  "tagged image",
			image: "myimage:abc",
			want:  "myimage",
		},
		{
			name:  "tagged image on ported registry",
			image: "example.com:5000/myimage:abc",
			want:  "example.com:5000/myimage",
		},
		{
			name:  "digest image",
			image: "myimage@sha256:abc",
			want:  "myimage",
		},
		{
			name:  "digest image on ported registry",
			image: "example.com:5000/myimage@sha256:abc",
			want:  "example.com:5000/myimage",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripImageTag(tt.image); got != tt.want {
				t.Errorf("stripImageTag() = %v, want %v", got, tt.want)
			}
		})
	}
}
