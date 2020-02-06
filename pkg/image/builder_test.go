package image

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.undefinedlabs.com/scopeagent"
)

func TestReleaseStartNum(t *testing.T) {
	test := scopeagent.StartTest(t)
	defer test.End()
	var err error
	var ref *ImageRef

	ref, err = imageRefImage("debian:2.5")
	require.NoError(t, err)
	require.Equal(t, &ImageRef{
		Domain: "docker.io",
		Name:   "docker.io/library/debian",
		Tag:    "2.5",
	}, ref)
	require.Equal(t,
		"docker-archive/docker.io/library/debian/2.5",
		ref.pathInBundle("docker-archive"))

	ref, err = imageRefImage("debian")
	require.NoError(t, err)
	require.Equal(t, &ImageRef{
		Domain: "docker.io",
		Name:   "docker.io/library/debian",
		Tag:    "latest",
	}, ref)
	require.Equal(t,
		"oci-archive/docker.io/library/debian/latest",
		ref.pathInBundle("oci-archive"))

	ref, err = imageRefImage("quay.io/replicated/debian:2.5")
	require.NoError(t, err)
	require.Equal(t, &ImageRef{
		Domain: "quay.io",
		Name:   "quay.io/replicated/debian",
		Tag:    "2.5",
	}, ref)
	require.Equal(t,
		"docker-archive/quay.io/replicated/debian/2.5",
		ref.pathInBundle("docker-archive"))

	ref, err = imageRefImage("myorg/ubuntu:14")
	require.NoError(t, err)
	require.Equal(t, &ImageRef{
		Domain: "docker.io",
		Name:   "docker.io/myorg/ubuntu",
		Tag:    "14",
	}, ref)
	require.Equal(t,
		"docker-archive/docker.io/myorg/ubuntu/14",
		ref.pathInBundle("docker-archive"))

	ref, err = imageRefImage("myorg/ubuntu@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2")
	require.NoError(t, err)
	require.Equal(t, &ImageRef{
		Domain: "docker.io",
		Name:   "docker.io/myorg/ubuntu",
		Digest: "sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2",
	}, ref)
	require.Equal(t,
		"docker-archive/docker.io/myorg/ubuntu/sha256/45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2",
		ref.pathInBundle("docker-archive"))
}
