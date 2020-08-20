package supportbundle

import (
	"testing"

	_ "go.undefinedlabs.com/scopeagent/autoinstrument"
)

func Test_refFromBundleID(t *testing.T) {
	// tests := []struct {
	// 	name           string
	// 	bundleID       string
	// 	storageBaseURI string
	// 	expect         string
	// }{
	// 	{
	// 		name:           "docker dist",
	// 		bundleID:       "a",
	// 		storageBaseURI: "docker://my-reg:5000",
	// 		expect:         "my-reg:5000/supportbundle:a",
	// 	},
	// 	{
	// 		name:           "docker dist with trailing slash",
	// 		bundleID:       "a",
	// 		storageBaseURI: "docker://my-reg:5000/",
	// 		expect:         "my-reg:5000/supportbundle:a",
	// 	},
	// 	{
	// 		name:           "with an org",
	// 		bundleID:       "a",
	// 		storageBaseURI: "docker://my-reg:5000/test/",
	// 		expect:         "my-reg:5000/test/supportbundle:a",
	// 	},
	// }
	// for _, test := range tests {
	// 	t.Run(test.name, func(t *testing.T) {
	// 		actual := refFromBundleID(test.bundleID, test.storageBaseURI)
	// 		assert.Equal(t, test.expect, actual)
	// 	})
	// }
}
