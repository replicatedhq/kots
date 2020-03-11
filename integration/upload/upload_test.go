package upload

import (
	"io/ioutil"
	"path"
	"testing"

	"github.com/replicatedhq/kots/pkg/auth"
	kotsupload "github.com/replicatedhq/kots/pkg/upload"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/stretchr/testify/require"
	_ "go.undefinedlabs.com/scopeagent/autoinstrument"
)

func Test_Upload(t *testing.T) {
	tests := []struct {
		path                 string
		namespace            string
		upstreamURI          string
		expectedUpdateCursor string
		expectedVersionLabel string
		expectedLicense      string
		newAppName           string
	}{
		{
			path:                 "kitchen-sink",
			namespace:            "default",
			upstreamURI:          "replicated://kitchen-sink",
			expectedUpdateCursor: "",
			expectedVersionLabel: "",
			expectedLicense:      "",
			newAppName:           "kitchen-sink",
		},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			req := require.New(t)
			auth.SetAuthSlugCache("Kots " + util.GenPassword(32))

			uploadOptions := kotsupload.UploadOptions{
				Namespace:       test.namespace,
				UpstreamURI:     test.upstreamURI,
				ExistingAppSlug: "",
				NewAppName:      test.newAppName,
				Endpoint:        "http://localhost:3001",
				Silent:          true,
			}

			expectedData, err := ioutil.ReadFile(path.Join("tests", test.path, "expected-archive.tar.gz"))
			req.NoError(err)

			method := "POST"
			stopCh, err := StartMockServer("http://localhost:3001", method, test.expectedUpdateCursor, test.expectedVersionLabel, test.expectedLicense, expectedData)
			req.NoError(err)

			defer func() {
				stopCh <- true
			}()

			err = kotsupload.Upload(path.Join("tests", test.path, "input"), uploadOptions)
			req.NoError(err)
		})
	}
}
