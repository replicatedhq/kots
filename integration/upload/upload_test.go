package upload

import (
	"io/ioutil"
	"path"
	"testing"

	kotsupload "github.com/replicatedhq/kots/pkg/upload"
	"github.com/stretchr/testify/require"
)

func Test_Upload(t *testing.T) {
	tests := []struct {
		path                 string
		expectedUpdateCursor string
		expectedVersionLabel string
		expectedLicense      string
	}{
		{
			path:                 "kitchen-sink",
			expectedUpdateCursor: "",
			expectedVersionLabel: "",
			expectedLicense:      "",
		},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			req := require.New(t)

			uploadOptions := kotsupload.UploadOptions{
				Namespace:       "",
				UpstreamURI:     "",
				ExistingAppSlug: "",
				NewAppName:      "",
				Endpoint:        "http://localhost:3000",
			}

			expectedData, err := ioutil.ReadFile(path.Join("tests", test.path, "expected-archive.tar.gz"))
			req.NoError(err)

			method := "POST"
			stopCh, err := StartMockServer("http://localhost:3000", method, test.expectedUpdateCursor, test.expectedVersionLabel, test.expectedLicense, expectedData)
			req.NoError(err)

			defer func() {
				stopCh <- true
			}()

			err = kotsupload.Upload(path.Join("tests", test.path, "input"), uploadOptions)
			req.NoError(err)
		})
	}
}
