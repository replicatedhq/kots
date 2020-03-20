package kotsadm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	_ "go.undefinedlabs.com/scopeagent/autoinstrument"
)

func Test_kotsadmTagForVersionString(t *testing.T) {
	tests := []struct {
		version string
		expect  string
	}{
		{
			version: "v1.13.9-57-g07f06e6-dirty",
			expect:  "alpha",
		},
		{
			version: "v1.14.0-beta",
			expect:  "v1.14.0-beta",
		},
		{
			version: "v1.14.0",
			expect:  "v1.14.0",
		},
		{
			version: "1.14.0",
			expect:  "v1.14.0",
		},
	}

	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			actual := kotsadmTagForVersionString(test.version)
			assert.Equal(t, test.expect, actual)
		})
	}
}
