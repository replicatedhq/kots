package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	_ "go.undefinedlabs.com/scopeagent/autoinstrument"
)

func Test_isHelmInstalled(t *testing.T) {
	// assert that we return false when not running in a cluster
	installed := isHelmInstalled()
	assert.Equal(t, false, installed)
}
