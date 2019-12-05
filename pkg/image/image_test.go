package image

import (
	"fmt"
	"testing"

	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/v3/pkg/image"
)

func Test_ImageNameFromNameParts(t *testing.T) {
	registryOps := registry.RegistryOptions{
		Endpoint:  "localhost:5000",
		Namespace: "somebigbank",
	}

	tests := []struct {
		name     string
		parts    []string
		expected image.Image
		isError  bool
	}{
		{
			name:     "bad name format",
			parts:    []string{"quay.io", "latest"},
			expected: image.Image{},
			isError:  true,
		},
		{
			name:  "ECR style image",
			parts: []string{"411111111111.dkr.ecr.us-west-1.amazonaws.com", "myrepo", "v0.0.1"},
			expected: image.Image{
				Name:    "411111111111.dkr.ecr.us-west-1.amazonaws.com/myrepo:v0.0.1",
				NewName: fmt.Sprintf("%s/%s/myrepo", registryOps.Endpoint, registryOps.Namespace),
				NewTag:  "v0.0.1",
				Digest:  "",
			},
			isError: false,
		},
		{
			name:  "four parts with tag",
			parts: []string{"quay.io", "someorg", "debian", "0.1"},
			expected: image.Image{
				Name:    "quay.io/someorg/debian:0.1",
				NewName: fmt.Sprintf("%s/%s/debian", registryOps.Endpoint, registryOps.Namespace),
				NewTag:  "0.1",
				Digest:  "",
			},
			isError: false,
		},
		{
			name:  "five parts with sha",
			parts: []string{"quay.io", "someorg", "debian", "sha256", "1234567890abcdef"},
			expected: image.Image{
				Name:    "quay.io/someorg/debian@sha256:1234567890abcdef",
				NewName: fmt.Sprintf("%s/%s/debian", registryOps.Endpoint, registryOps.Namespace),
				NewTag:  "",
				Digest:  "1234567890abcdef",
			},
			isError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			image, err := ImageInfoFromFile(registryOps, test.parts)
			if test.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, image)
			}
		})
	}
}

func TestDestImageName(t *testing.T) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DestImageName(tt.args.registry, tt.args.srcImage); got != tt.want {
				t.Errorf("DestImageName() = %v, want %v", got, tt.want)
			}
		})
	}
}
