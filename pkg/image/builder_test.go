package image

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReleaseStartNum(t *testing.T) {
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
		"docker.io/library/debian/2.5",
		ref.pathInBundle())

	ref, err = imageRefImage("debian")
	require.NoError(t, err)
	require.Equal(t, &ImageRef{
		Domain: "docker.io",
		Name:   "docker.io/library/debian",
		Tag:    "latest",
	}, ref)
	require.Equal(t,
		"docker.io/library/debian/latest",
		ref.pathInBundle())

	ref, err = imageRefImage("quay.io/replicated/debian:2.5")
	require.NoError(t, err)
	require.Equal(t, &ImageRef{
		Domain: "quay.io",
		Name:   "quay.io/replicated/debian",
		Tag:    "2.5",
	}, ref)
	require.Equal(t,
		"quay.io/replicated/debian/2.5",
		ref.pathInBundle())

	ref, err = imageRefImage("myorg/ubuntu:14")
	require.NoError(t, err)
	require.Equal(t, &ImageRef{
		Domain: "docker.io",
		Name:   "docker.io/myorg/ubuntu",
		Tag:    "14",
	}, ref)
	require.Equal(t,
		"docker.io/myorg/ubuntu/14",
		ref.pathInBundle())

	ref, err = imageRefImage("myorg/ubuntu@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2")
	require.NoError(t, err)
	require.Equal(t, &ImageRef{
		Domain: "docker.io",
		Name:   "docker.io/myorg/ubuntu",
		Digest: "sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2",
	}, ref)
	require.Equal(t,
		"docker.io/myorg/ubuntu/sha256/45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2",
		ref.pathInBundle())
}
