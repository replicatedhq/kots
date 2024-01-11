package replicated

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mholt/archiver/v3"
	"github.com/replicatedhq/kots/integration/util"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/store"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
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

	testDirs, err := ioutil.ReadDir("tests")
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

			archiveData, err := os.ReadFile(path.Join(testResourcePath, "archive.tar.gz"))
			req.NoError(err)

			licenseFilepath := path.Join(testResourcePath, "license.yaml")
			licenseFile, err := os.ReadFile(licenseFilepath)
			req.NoError(err)

			server, err := StartMockServer(archiveData, licenseFile)
			req.NoError(err)
			defer server.Close()

			actualDir := t.TempDir()

			pullOptions := pull.PullOptions{
				RootDir:                 actualDir,
				LicenseFile:             licenseFilepath,
				Namespace:               namespace,
				LicenseEndpointOverride: "http://localhost:3000",
				ExcludeAdminConsole:     true,
				ExcludeKotsKinds:        true,
				Silent:                  true,
			}
			_, err = pull.Pull("replicated://integration", pullOptions)
			req.NoError(err)

			// create an archive of the actual results
			actualFilesystemDir := t.TempDir()

			paths := []string{
				path.Join(actualDir, "upstream"),
				path.Join(actualDir, "base"),
				path.Join(actualDir, "overlays"),
			}

			tarGz := archiver.TarGz{
				Tar: &archiver.Tar{
					ImplicitTopLevelFolder: false,
				},
			}
			err = tarGz.Archive(paths, path.Join(actualFilesystemDir, "archive.tar.gz"))
			req.NoError(err)

			actualFilesystemBytes, err := os.ReadFile(path.Join(actualFilesystemDir, "archive.tar.gz"))
			req.NoError(err)

			// create an archive of the expected
			expectedFilesystemDir := t.TempDir()

			paths = []string{
				path.Join(testResourcePath, "expected", "upstream"),
				path.Join(testResourcePath, "expected", "base"),
				path.Join(testResourcePath, "expected", "overlays"),
			}
			err = tarGz.Archive(paths, path.Join(expectedFilesystemDir, "archive.tar.gz"))
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
