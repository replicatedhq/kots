package imageworker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseImageName(t *testing.T) {
	url, image, tag, err := parseImageName("redis:4")
	require.NoError(t, err)
	assert.Equal(t, "index.docker.io", url)
	assert.Equal(t, "library/redis", image)
	assert.Equal(t, "4", tag)

	url, image, tag, err = parseImageName("k8s.gcr.io/cluster-proportional-autoscaler-amd64:1.1.2-r2")
	require.NoError(t, err)
	assert.Equal(t, "k8s.gcr.io", url)
	assert.Equal(t, "library/cluster-proportional-autoscaler-amd64", image)
	assert.Equal(t, "1.1.2-r2", tag)

	url, image, tag, err = parseImageName("quay.io/coreos/grafana-watcher:v0.0.8")
	require.NoError(t, err)
	assert.Equal(t, "quay.io", url)
	assert.Equal(t, "coreos/grafana-watcher", image)
	assert.Equal(t, "v0.0.8", tag)

	url, image, tag, err = parseImageName("grafana/grafana:5.0.1")
	require.NoError(t, err)
	assert.Equal(t, "index.docker.io", url)
	assert.Equal(t, "grafana/grafana", image)
	assert.Equal(t, "5.0.1", tag)

	url, image, tag, err = parseImageName("postgres:10.0")
	require.NoError(t, err)
	assert.Equal(t, "index.docker.io", url)
	assert.Equal(t, "library/postgres", image)
	assert.Equal(t, "10.0", tag)
}
