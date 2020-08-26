package ocistore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	_ "go.undefinedlabs.com/scopeagent/autoinstrument"
)

func Test_refFromAppVersion(t *testing.T) {
	tests := []struct {
		name           string
		appID          string
		sequence       int64
		storageBaseURI string
		expect         string
	}{
		{
			name:           "docker dist",
			appID:          "a",
			sequence:       0,
			storageBaseURI: "docker://my-reg:5000",
			expect:         "my-reg:5000/a:0",
		},
		{
			name:           "docker dist with trailing slash",
			appID:          "a",
			sequence:       0,
			storageBaseURI: "docker://my-reg:5000/",
			expect:         "my-reg:5000/a:0",
		},
		{
			name:           "with org / project / whatever it's called",
			appID:          "a",
			sequence:       0,
			storageBaseURI: "docker://my-reg:5000/my-app",
			expect:         "my-reg:5000/my-app/a:0",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := refFromAppVersion(test.appID, test.sequence, test.storageBaseURI)
			assert.Equal(t, test.expect, actual)
		})
	}
}
