package replicated

import (
	"os"
	"path"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/kots/integration/util"
	"github.com/replicatedhq/kots/pkg/archiveutil"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/store"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	"github.com/replicatedhq/kotskinds/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_PullReplicated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock_store.NewMockStore(ctrl)
	store.SetStore(mockStore)
	defer store.SetStore(nil)

	mockStore.EXPECT().ListInstalledApps().MaxTimes(1)

	namespace := "test_ns"

	testDirs, err := os.ReadDir("tests")
	if err != nil {
		panic(err)
	}

	for _, testDir := range testDirs {
		if !testDir.IsDir() {
			continue
		}

		testResourcePath := path.Join("tests", testDir.Name())

		t.Run(testDir.Name(), func(t *testing.T) {
			req := require.New(t)

			// Set up custom global key for v1beta2 test licenses only
			// This key was used to sign the v1beta2 test licenses in testdata/
			if testDir.Name() == "kitchen-sink-v1beta2" {
				globalKey := `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxHh2OXzDqlQ7kZJ1d4zr
wbpXsSFHcYzr+k6pe+QXLUelAMvlik9NXauIt+YFtEAxNypV+xPCr8ClH5L2qPPb
QBeG0ExxzvRshDMGxm7TXVHzTXQCrD7azS8Va6RsAB4tJMlvymn2uHsQDbShQiOY
RKaRY/KKBmaIcYmysaSvfU8E5Ve9f4478X3u1cPzKUG6dk5j1Nt3nSv3BWINM5ec
IXJQCB+gQVkOjzvA9aRVtLJtFqAoX7A6BfTNqrx35eyBEmzQOo0Mx1JkZDDW4+qC
bhC0kq14IRpwKFIALBhSojfbJelM+gCv3wjF4hrWxAZQzWSPexP1Msof2KbrniEe
LQIDAQAB
-----END PUBLIC KEY-----
`
				if err := crypto.SetCustomPublicKeyRSA(globalKey); err != nil {
					t.Fatalf("failed to set custom global key for v1beta2 tests: %v", err)
				}
				defer crypto.ResetCustomPublicKeyRSA() // Clean up after test
			}

			archiveData, err := os.ReadFile(path.Join(testResourcePath, "archive.tar.gz"))
			req.NoError(err)

			licenseFilepath := path.Join(testResourcePath, "license.yaml")
			licenseFile, err := os.ReadFile(licenseFilepath)
			req.NoError(err)

			server, err := StartMockServer(archiveData, licenseFile)
			req.NoError(err)
			defer server.Close()

			actualDir := t.TempDir()

			// Use different channel ID for v1beta2 test to match its license
			selectedChannelID := "1vusIYZLAVxMG6q760OJmRKj5i5"
			if testDir.Name() == "kitchen-sink-v1beta2" {
				selectedChannelID = "1"
			}

			pullOptions := pull.PullOptions{
				RootDir:                 actualDir,
				LicenseFile:             licenseFilepath,
				Namespace:               namespace,
				LicenseEndpointOverride: "http://localhost:3000",
				ExcludeAdminConsole:     true,
				ExcludeKotsKinds:        true,
				Silent:                  true,
				AppSelectedChannelID:    selectedChannelID,
			}
			_, err = pull.Pull("replicated://integration", pullOptions)
			req.NoError(err)

			// create an archive of the actual results
			actualFilesystemDir := t.TempDir()

			filepaths := map[string]string{
				path.Join(actualDir, "upstream"): "",
				path.Join(actualDir, "base"):     "",
				path.Join(actualDir, "overlays"): "",
			}

			err = archiveutil.CreateTGZ(t.Context(), filepaths, path.Join(actualFilesystemDir, "archive.tar.gz"))
			req.NoError(err)

			actualFilesystemBytes, err := os.ReadFile(path.Join(actualFilesystemDir, "archive.tar.gz"))
			req.NoError(err)

			// create an archive of the expected
			expectedFilesystemDir := t.TempDir()

			filepaths = map[string]string{
				path.Join(testResourcePath, "expected", "upstream"): "",
				path.Join(testResourcePath, "expected", "base"):     "",
				path.Join(testResourcePath, "expected", "overlays"): "",
			}
			err = archiveutil.CreateTGZ(t.Context(), filepaths, path.Join(expectedFilesystemDir, "archive.tar.gz"))
			req.NoError(err)

			expectedFilesystemBytes, err := os.ReadFile(path.Join(expectedFilesystemDir, "archive.tar.gz"))
			req.NoError(err)

			compareOptions := util.CompareOptions{
				IgnoreFilesInActual: []string{path.Join("upstream", "userdata", "license.yaml")},
			}

			ok, err := util.CompareTars(expectedFilesystemBytes, actualFilesystemBytes, compareOptions)
			req.NoError(err)

			assert.Equal(t, true, ok)
		})
	}
}
