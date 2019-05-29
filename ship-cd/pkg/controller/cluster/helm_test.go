package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_isHelmInstalled(t *testing.T) {
	// assert that we return false when not running in a cluster
	installed := isHelmInstalled()
	assert.Equal(t, false, installed)
}
