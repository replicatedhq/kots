package imageutil

import (
	"fmt"
	"reflect"
	"testing"

	registrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

func Test_RewriteDockerArchiveImage(t *testing.T) {
	tests := []struct {
		name        string
		parts       []string
		registryOps registrytypes.RegistryOptions
		expected    kustomizetypes.Image
		isError     bool
	}{
		{
			name:  "bad name format",
			parts: []string{"quay.io", "latest"},
			registryOps: registrytypes.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somebigbank",
			},
			expected: kustomizetypes.Image{},
			isError:  true,
		},
		{
			name:  "ECR style image",
			parts: []string{"411111111111.dkr.ecr.us-west-1.amazonaws.com", "myrepo", "v0.0.1"},
			registryOps: registrytypes.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somebigbank",
			},
			expected: kustomizetypes.Image{
				Name:    "411111111111.dkr.ecr.us-west-1.amazonaws.com/myrepo",
				NewName: "localhost:5000/somebigbank/myrepo",
				NewTag:  "v0.0.1",
				Digest:  "",
			},
			isError: false,
		},
		{
			name:  "four parts with tag",
			parts: []string{"quay.io", "someorg", "debian", "0.1"},
			registryOps: registrytypes.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somebigbank",
			},
			expected: kustomizetypes.Image{
				Name:    "quay.io/someorg/debian",
				NewName: "localhost:5000/somebigbank/debian",
				NewTag:  "0.1",
				Digest:  "",
			},
			isError: false,
		},
		{
			name:  "five parts with sha",
			parts: []string{"quay.io", "someorg", "debian", "sha256", "1234567890abcdef"},
			registryOps: registrytypes.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somebigbank",
			},
			expected: kustomizetypes.Image{
				Name:    "quay.io/someorg/debian",
				NewName: "localhost:5000/somebigbank/debian",
				NewTag:  "",
				Digest:  "sha256:1234567890abcdef",
			},
			isError: false,
		},
		{
			name:  "no namespace",
			parts: []string{"quay.io", "someorg", "debian", "0.1"},
			registryOps: registrytypes.RegistryOptions{
				Endpoint: "localhost:5000",
			},
			expected: kustomizetypes.Image{
				Name:    "quay.io/someorg/debian",
				NewName: "localhost:5000/debian",
				NewTag:  "0.1",
				Digest:  "",
			},
			isError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			image, err := RewriteDockerArchiveImage(test.registryOps, test.parts)
			if test.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, image)
			}
		})
	}
}

func TestRewriteDockerRegistryImage(t *testing.T) {
	type args struct {
		srcImage     string
		destRegistry registrytypes.RegistryOptions
	}

	tests := []struct {
		name    string
		args    args
		want    *kustomizetypes.Image
		wantErr bool
	}{
		{
			name: "no tag or digest or namespace",
			args: args{
				srcImage: "alpine",
				destRegistry: registrytypes.RegistryOptions{
					Endpoint: "private.registry.com",
				},
			},
			want: &kustomizetypes.Image{
				Name:    "alpine",
				NewName: "private.registry.com/alpine",
				NewTag:  "latest",
			},
			wantErr: false,
		},
		{
			name: "no tag or digest with namespace",
			args: args{
				srcImage: "alpine",
				destRegistry: registrytypes.RegistryOptions{
					Endpoint:  "private.registry.com",
					Namespace: "replicatedcom",
				},
			},
			want: &kustomizetypes.Image{
				Name:    "alpine",
				NewName: "private.registry.com/replicatedcom/alpine",
				NewTag:  "latest",
			},
			wantErr: false,
		},
		{
			name: "tag only no namespace",
			args: args{
				srcImage: "alpine:3.14",
				destRegistry: registrytypes.RegistryOptions{
					Endpoint: "private.registry.com",
				},
			},
			want: &kustomizetypes.Image{
				Name:    "alpine",
				NewName: "private.registry.com/alpine",
				NewTag:  "3.14",
			},
			wantErr: false,
		},
		{
			name: "tag only with namespace",
			args: args{
				srcImage: "alpine:3.14",
				destRegistry: registrytypes.RegistryOptions{
					Endpoint:  "private.registry.com",
					Namespace: "replicatedcom",
				},
			},
			want: &kustomizetypes.Image{
				Name:    "alpine",
				NewName: "private.registry.com/replicatedcom/alpine",
				NewTag:  "3.14",
			},
			wantErr: false,
		},
		{
			name: "digest only no namespace",
			args: args{
				srcImage: "alpine@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
				destRegistry: registrytypes.RegistryOptions{
					Endpoint: "private.registry.com",
				},
			},
			want: &kustomizetypes.Image{
				Name:    "alpine",
				NewName: "private.registry.com/alpine",
				Digest:  "sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			},
			wantErr: false,
		},
		{
			name: "digest only with namespace",
			args: args{
				srcImage: "alpine@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
				destRegistry: registrytypes.RegistryOptions{
					Endpoint:  "private.registry.com",
					Namespace: "replicatedcom",
				},
			},
			want: &kustomizetypes.Image{
				Name:    "alpine",
				NewName: "private.registry.com/replicatedcom/alpine",
				Digest:  "sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			},
			wantErr: false,
		},
		{
			name: "tag and digest no namespace",
			args: args{
				srcImage: "alpine:3.14@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
				destRegistry: registrytypes.RegistryOptions{
					Endpoint: "private.registry.com",
				},
			},
			want: &kustomizetypes.Image{
				Name:    "alpine",
				NewName: "private.registry.com/alpine",
				Digest:  "sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			},
			wantErr: false,
		},
		{
			name: "tag and digest with namespace",
			args: args{
				srcImage: "alpine:3.14@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
				destRegistry: registrytypes.RegistryOptions{
					Endpoint:  "private.registry.com",
					Namespace: "replicatedcom",
				},
			},
			want: &kustomizetypes.Image{
				Name:    "alpine",
				NewName: "private.registry.com/replicatedcom/alpine",
				Digest:  "sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			},
			wantErr: false,
		},
		{
			name: "tag and digest with multipart namespace",
			args: args{
				srcImage: "alpine:3.14@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
				destRegistry: registrytypes.RegistryOptions{
					Endpoint:  "private.registry.com",
					Namespace: "replicatedcom/test",
				},
			},
			want: &kustomizetypes.Image{
				Name:    "alpine",
				NewName: "private.registry.com/replicatedcom/test/alpine",
				Digest:  "sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			},
			wantErr: false,
		},
		{
			name: "private image - no tag or digest or namespace",
			args: args{
				srcImage: "quay.io/replicatedhq/alpine",
				destRegistry: registrytypes.RegistryOptions{
					Endpoint: "private.registry.com",
				},
			},
			want: &kustomizetypes.Image{
				Name:    "quay.io/replicatedhq/alpine",
				NewName: "private.registry.com/alpine",
				NewTag:  "latest",
			},
			wantErr: false,
		},
		{
			name: "private image - no tag or digest with namespace",
			args: args{
				srcImage: "quay.io/replicatedhq/alpine",
				destRegistry: registrytypes.RegistryOptions{
					Endpoint:  "private.registry.com",
					Namespace: "replicatedcom",
				},
			},
			want: &kustomizetypes.Image{
				Name:    "quay.io/replicatedhq/alpine",
				NewName: "private.registry.com/replicatedcom/alpine",
				NewTag:  "latest",
			},
			wantErr: false,
		},
		{
			name: "private image - tag only no namespace",
			args: args{
				srcImage: "quay.io/replicatedhq/alpine:3.14",
				destRegistry: registrytypes.RegistryOptions{
					Endpoint: "private.registry.com",
				},
			},
			want: &kustomizetypes.Image{
				Name:    "quay.io/replicatedhq/alpine",
				NewName: "private.registry.com/alpine",
				NewTag:  "3.14",
			},
			wantErr: false,
		},
		{
			name: "private image - tag only with namespace",
			args: args{
				srcImage: "quay.io/replicatedhq/alpine:3.14",
				destRegistry: registrytypes.RegistryOptions{
					Endpoint:  "private.registry.com",
					Namespace: "replicatedcom",
				},
			},
			want: &kustomizetypes.Image{
				Name:    "quay.io/replicatedhq/alpine",
				NewName: "private.registry.com/replicatedcom/alpine",
				NewTag:  "3.14",
			},
			wantErr: false,
		},
		{
			name: "private image - digest only no namespace",
			args: args{
				srcImage: "quay.io/replicatedhq/alpine@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
				destRegistry: registrytypes.RegistryOptions{
					Endpoint: "private.registry.com",
				},
			},
			want: &kustomizetypes.Image{
				Name:    "quay.io/replicatedhq/alpine",
				NewName: "private.registry.com/alpine",
				Digest:  "sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			},
			wantErr: false,
		},
		{
			name: "private image - digest only with namespace",
			args: args{
				srcImage: "quay.io/replicatedhq/alpine@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
				destRegistry: registrytypes.RegistryOptions{
					Endpoint:  "private.registry.com",
					Namespace: "replicatedcom",
				},
			},
			want: &kustomizetypes.Image{
				Name:    "quay.io/replicatedhq/alpine",
				NewName: "private.registry.com/replicatedcom/alpine",
				Digest:  "sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			},
			wantErr: false,
		},
		{
			name: "private image - tag and digest no namespace",
			args: args{
				srcImage: "quay.io/replicatedhq/alpine:3.14@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
				destRegistry: registrytypes.RegistryOptions{
					Endpoint: "private.registry.com",
				},
			},
			want: &kustomizetypes.Image{
				Name:    "quay.io/replicatedhq/alpine",
				NewName: "private.registry.com/alpine",
				Digest:  "sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			},
			wantErr: false,
		},
		{
			name: "private image - tag and digest with namespace",
			args: args{
				srcImage: "quay.io/replicatedhq/alpine:3.14@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
				destRegistry: registrytypes.RegistryOptions{
					Endpoint:  "private.registry.com",
					Namespace: "replicatedcom",
				},
			},
			want: &kustomizetypes.Image{
				Name:    "quay.io/replicatedhq/alpine",
				NewName: "private.registry.com/replicatedcom/alpine",
				Digest:  "sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			},
			wantErr: false,
		},
		{
			name: "private image - tag and digest with multipart namespace",
			args: args{
				srcImage: "quay.io/replicatedhq/alpine:3.14@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
				destRegistry: registrytypes.RegistryOptions{
					Endpoint:  "private.registry.com",
					Namespace: "replicatedcom/test",
				},
			},
			want: &kustomizetypes.Image{
				Name:    "quay.io/replicatedhq/alpine",
				NewName: "private.registry.com/replicatedcom/test/alpine",
				Digest:  "sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RewriteDockerRegistryImage(tt.args.destRegistry, tt.args.srcImage)
			if (err != nil) != tt.wantErr {
				t.Errorf("RewriteDockerRegistryImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RewriteDockerRegistryImage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_DestImage(t *testing.T) {
	registryOps := registrytypes.RegistryOptions{
		Endpoint:  "localhost:5000",
		Namespace: "somebigbank",
	}

	type args struct {
		registry registrytypes.RegistryOptions
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
				srcImage: "quay.io/someorg/debian@sha256:17c5f462c92fc39303e6363c65e074559f8d6a1354150027ed5053557e3298c5",
			},
			want: fmt.Sprintf("%s/%s/debian@sha256:17c5f462c92fc39303e6363c65e074559f8d6a1354150027ed5053557e3298c5", registryOps.Endpoint, registryOps.Namespace),
		},
		{
			name: "Image with tag and digest",
			args: args{
				registry: registryOps,
				srcImage: "quay.io/someorg/debian:0.1@sha256:17c5f462c92fc39303e6363c65e074559f8d6a1354150027ed5053557e3298c5",
			},
			want: fmt.Sprintf("%s/%s/debian@sha256:17c5f462c92fc39303e6363c65e074559f8d6a1354150027ed5053557e3298c5", registryOps.Endpoint, registryOps.Namespace),
		},
		{
			name: "No Namespace",
			args: args{
				registry: registrytypes.RegistryOptions{
					Endpoint: "localhost:5000",
				},
				srcImage: "quay.io/someorg/debian:0.1",
			},
			want: fmt.Sprintf("%s/debian:0.1", registryOps.Endpoint),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			got, err := DestImage(tt.args.registry, tt.args.srcImage)
			req.NoError(err)

			if got != tt.want {
				t.Errorf("DestImageName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDestImageFromKustomizeImage(t *testing.T) {
	type args struct {
		image kustomizetypes.Image
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "latest tag",
			args: args{
				image: kustomizetypes.Image{
					NewName: "private.registry.com/replicatedcom/alpine",
					NewTag:  "latest",
				},
			},
			want: "private.registry.com/replicatedcom/alpine:latest",
		},
		{
			name: "tag only",
			args: args{
				image: kustomizetypes.Image{
					NewName: "private.registry.com/replicatedcom/alpine",
					NewTag:  "3.14",
				},
			},
			want: "private.registry.com/replicatedcom/alpine:3.14",
		},
		{
			name: "digest only",
			args: args{
				image: kustomizetypes.Image{
					NewName: "private.registry.com/replicatedcom/alpine",
					Digest:  "sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
				},
			},
			want: "private.registry.com/replicatedcom/alpine@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
		},
		{
			name: "tag and digest",
			args: args{
				image: kustomizetypes.Image{
					NewName: "private.registry.com/replicatedcom/alpine",
					NewTag:  "3.14",
					Digest:  "sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
				},
			},
			want: "private.registry.com/replicatedcom/alpine@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DestImageFromKustomizeImage(tt.args.image); got != tt.want {
				t.Errorf("DestImageFromKustomizeImage() = %v, want %v", got, tt.want)
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
				{
					Name:    "docker.io/image:latest",
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

			got, err := BuildImageAltNames(tt.rewrittenImage)
			req.NoError(err)
			req.Equal(tt.want, got)
		})
	}
}

func Test_KustomizeImage(t *testing.T) {
	tests := []struct {
		name         string
		destRegistry registrytypes.RegistryOptions
		image        string
		want         []kustomizetypes.Image
	}{
		{
			name: "naked image",
			destRegistry: registrytypes.RegistryOptions{
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
				{
					Name:    "docker.io/redis",
					NewName: "localhost:5000/somebigbank/redis",
					NewTag:  "latest",
					Digest:  "",
				},
			},
		},
		{
			name: "naked tagged image",
			destRegistry: registrytypes.RegistryOptions{
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
				{
					Name:    "docker.io/redis",
					NewName: "localhost:5000/somebigbank/redis",
					NewTag:  "v1",
					Digest:  "",
				},
			},
		},
		{
			name: "naked contentAddressableSha image",
			destRegistry: registrytypes.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somesmallcorp",
			},
			image: "redis@sha256:ae39a6f5c07297d7ab64dbd4f82c77c874cc6a94cea29fdec309d0992574b4f7",
			want: []kustomizetypes.Image{
				{
					Name:    "redis",
					NewName: "localhost:5000/somesmallcorp/redis",
					NewTag:  "",
					Digest:  "sha256:ae39a6f5c07297d7ab64dbd4f82c77c874cc6a94cea29fdec309d0992574b4f7",
				},
				{
					Name:    "docker.io/library/redis",
					NewName: "localhost:5000/somesmallcorp/redis",
					NewTag:  "",
					Digest:  "sha256:ae39a6f5c07297d7ab64dbd4f82c77c874cc6a94cea29fdec309d0992574b4f7",
				},
				{
					Name:    "library/redis",
					NewName: "localhost:5000/somesmallcorp/redis",
					NewTag:  "",
					Digest:  "sha256:ae39a6f5c07297d7ab64dbd4f82c77c874cc6a94cea29fdec309d0992574b4f7",
				},
				{
					Name:    "docker.io/redis",
					NewName: "localhost:5000/somesmallcorp/redis",
					NewTag:  "",
					Digest:  "sha256:ae39a6f5c07297d7ab64dbd4f82c77c874cc6a94cea29fdec309d0992574b4f7",
				},
			},
		},
		{
			name: "tagged image",
			destRegistry: registrytypes.RegistryOptions{
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
					Name:    "docker.io/redis",
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
			destRegistry: registrytypes.RegistryOptions{
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
			name: "quay.io tagged and digested image",
			destRegistry: registrytypes.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somebigbank",
			},
			image: "quay.io/library/alpine:3.14@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			want: []kustomizetypes.Image{
				{
					Name:    "quay.io/library/alpine",
					NewName: "localhost:5000/somebigbank/alpine",
					NewTag:  "",
					Digest:  "sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
				},
			},
		},
		{
			name: "ported registry tagged image",
			destRegistry: registrytypes.RegistryOptions{
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
			destRegistry: registrytypes.RegistryOptions{
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
			destRegistry: registrytypes.RegistryOptions{
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
			name: "ecr",
			destRegistry: registrytypes.RegistryOptions{
				Endpoint:  "localhost:5000",
				Namespace: "somebigbank",
			},
			image: "111122222333.dkr.ecr.us-east-1.amazonaws.com/frontend:v1.0.1",
			want: []kustomizetypes.Image{
				{
					Name:    "111122222333.dkr.ecr.us-east-1.amazonaws.com/frontend",
					NewName: "localhost:5000/somebigbank/frontend",
					NewTag:  "v1.0.1",
					Digest:  "",
				},
			},
		},
		{
			name: "no namespace",
			destRegistry: registrytypes.RegistryOptions{
				Endpoint: "localhost:5000",
			},
			image: "docker.io/redis:v1",
			want: []kustomizetypes.Image{
				{
					Name:    "docker.io/redis",
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
				{
					Name:    "library/redis",
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
			got, err := KustomizeImage(tt.destRegistry, tt.image)
			req.NoError(err)
			req.Equal(tt.want, got)
		})
	}
}

func Test_StripImageTagAndDigest(t *testing.T) {
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
		{
			name:  "tagged and digest image",
			image: "myimage:1@sha256:abc",
			want:  "myimage",
		},
		{
			name:  "tagged and digest image on ported registry",
			image: "example.com:5000/myimage:1@sha256:abc",
			want:  "example.com:5000/myimage",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripImageTagAndDigest(tt.image); got != tt.want {
				t.Errorf("StripImageTagAndDigest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetTag(t *testing.T) {
	tt := []struct {
		name        string
		imageRef    string
		expectedTag string
		wantErr     bool
	}{
		{
			name:        "happy path",
			imageRef:    "some/image:v1.2.3",
			expectedTag: "v1.2.3",
		},
		{
			name:     "failed case",
			imageRef: "",
			wantErr:  true,
		},
		{
			name:     "no tag",
			imageRef: "foo/bar",
			wantErr:  true,
		},
		{
			name:     "fat fingered",
			imageRef: "some/image:",
			wantErr:  true,
		},
		{
			name:        "long image",
			imageRef:    "some/image/image2:v1.2.3",
			expectedTag: "v1.2.3",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := GetTag(tc.imageRef)
			if tc.wantErr {
				require.NotNil(t, err)
				return
			}
			require.Nil(t, err)
			require.Equal(t, tc.expectedTag, actual)
		})
	}
}

func Test_GetImageName(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		expected string
	}{
		{
			name:     "image with tag",
			image:    "quay.io/someorg/myimage:0.1",
			expected: "myimage",
		},
		{
			name:     "image with digest",
			image:    "quay.io/someorg/myimage@sha256:1234567890abcdef",
			expected: "myimage",
		},
		{
			name:     "image without tag or digest",
			image:    "quay.io/someorg/myimage",
			expected: "myimage",
		},
		{
			name:     "image with tag and digest",
			image:    "quay.io/someorg/myimage:0.1@sha256:1234567890abcdef",
			expected: "myimage",
		},
		{
			name:     "image with registry and port",
			image:    "myregistry.com:5000/someorg/myimage:0.1",
			expected: "myimage",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := GetImageName(test.image)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestChangeImageTag(t *testing.T) {
	tests := []struct {
		name   string
		image  string
		newTag string
		want   string
	}{
		{
			name:   "valid image with tag",
			image:  "myregistry.com/myimage:oldtag",
			newTag: "newtag",
			want:   "myregistry.com/myimage:newtag",
		},
		{
			name:   "valid image without tag",
			image:  "myregistry.com/myimage",
			newTag: "newtag",
			want:   "myregistry.com/myimage:newtag",
		},
		{
			name:   "valid image with digest",
			image:  "myregistry.com/myimage@sha256:a3e387f1517c3629c2a2513591c60d22320548762f06270d085f668dbdb9c5d4",
			newTag: "newtag",
			want:   "myregistry.com/myimage@sha256:a3e387f1517c3629c2a2513591c60d22320548762f06270d085f668dbdb9c5d4",
		},
		{
			name:   "valid image with tag and digest - not yet supported",
			image:  "myregistry.com/myimage:oldtag@sha256:a3e387f1517c3629c2a2513591c60d22320548762f06270d085f668dbdb9c5d4",
			newTag: "newtag",
			want:   "myregistry.com/myimage:oldtag@sha256:a3e387f1517c3629c2a2513591c60d22320548762f06270d085f668dbdb9c5d4",
		},
		{
			name:   "registry with a port",
			image:  "myregistry.com:5000/myimage:oldtag",
			newTag: "newtag",
			want:   "myregistry.com:5000/myimage:newtag",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ChangeImageTag(test.image, test.newTag)
			require.NoError(t, err)
			assert.Equal(t, test.want, result)
		})
	}
}

func TestSanitizeTag(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Valid repos
		{"1.0.1", "1.0.1"},
		{"my-App123", "my-App123"},
		{"123-456", "123-456"},
		{"my-App123.-", "my-App123.-"},
		{"my-App123-.", "my-App123-."},

		// Invalid repos
		{".invalid", "invalid"},
		{"-invalid", "invalid"},
		{"not valid!", "notvalid"},

		// Tags longer than 128 characters
		{"0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789", "01234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			sanitized := SanitizeTag(test.input)
			if sanitized != test.expected {
				t.Errorf("got: %s, expected: %s", sanitized, test.expected)
			}
		})
	}
}

func TestSanitizeRepo(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Valid repos
		{"nginx", "nginx"},
		{"my-app-123", "my-app-123"},
		{"my_app_123", "my_app_123"},
		{"charts.tar.gz", "charts.tar.gz"},

		// Invalid repos
		{"My-App-123", "my-app-123"},
		{"-invalid", "invalid"},
		{"_invalid", "invalid"},
		{".invalid", "invalid"},
		{"not valid!", "notvalid"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			sanitized := SanitizeRepo(test.input)
			if sanitized != test.expected {
				t.Errorf("got: %s, expected: %s", sanitized, test.expected)
			}
		})
	}
}
