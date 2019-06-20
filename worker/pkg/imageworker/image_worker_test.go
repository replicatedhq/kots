package imageworker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// func TestFetchTags(t *testing.T) {
// 	hostname := "index.docker.io"
// 	imageName := "library/redis"

// 	reg, err := initRegistryClient(hostname)
// 	require.NoError(t, err)

// 	tags, err := fetchTags(reg, imageName)
// 	require.NoError(t, err)
// 	assert.Contains(t, tags, "4")
// 	assert.Contains(t, tags, "4.0.0-alpine")
// }

func TestParseTags(t *testing.T) {
	tags := []string{
		"1.0.0",
		"latest",
	}
	semver, non := parseTags(tags)
	assert.Contains(t, non, "latest")
	assert.Equal(t, "1.0.0", semver[0].Original())
}
