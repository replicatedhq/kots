package replicated

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/integration/replicated/pull"
	"github.com/replicatedhq/kots/integration/util"
	kotspull "github.com/replicatedhq/kots/pkg/pull"
	"github.com/stretchr/testify/require"
)

func Test_PullReplicated(t *testing.T) {
	tests := []struct {
		name      string
		testDir   string
		licenseID string
	}{
		{
			name:      "kitchen sink",
			testDir:   "kitchen-sink",
			licenseID: "integration",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			licenseFile, err := ioutil.TempFile("", "license")
			req.NoError(err)

			// license := kotsv1beta1.License{}

			licenseData := []byte("")

			err = ioutil.WriteFile(licenseFile.Name(), licenseData, 0644)
			req.NoError(err)

			defer os.Remove(licenseFile.Name())
		})
	}
}

func runPullTests() error {
	pullTests := pull.ReplicatedPullTests()
	for _, pullTest := range pullTests {
		fmt.Printf("%s\n", pullTest.Name)

		licenseFile, err := ioutil.TempFile("", "license")
		if err != nil {
			return errors.Wrap(err, "failed to create tmp file")
		}
		err = ioutil.WriteFile(licenseFile.Name(), []byte(pullTest.LicenseData), 0644)
		if err != nil {
			return errors.Wrap(err, "failed to write license file")
		}
		defer os.Remove(licenseFile.Name())

		decoded, err := base64.StdEncoding.DecodeString(pullTest.ReplicatedAppArchive)
		if err != nil {
			return errors.Wrap(err, "failed to decode app archive")
		}

		stopCh, err := pull.StartMockServer(endpoint, "integration", "integration", decoded)
		if err != nil {
			return errors.Wrap(err, "failed to start mock server")
		}

		defer func() {
			stopCh <- true
		}()

		testDir, err := ioutil.TempDir("", "kots")
		if err != nil {
			return errors.Wrap(err, "failed to create temp dir")
		}
		defer os.RemoveAll(testDir)

		pullOptions := kotspull.PullOptions{
			RootDir:             testDir,
			LicenseFile:         licenseFile.Name(),
			ExcludeAdminConsole: true,
			ExcludeKotsKinds:    true,
			Silent:              true,
		}
		_, err = kotspull.Pull("replicated://integration", pullOptions)
		if err != nil {
			return errors.Wrap(err, "failed to pull")
		}

		actualFilesystemDir, err := ioutil.TempDir("", "kots")
		if err != nil {
			return errors.Wrap(err, "failed to create temp dir")
		}
		defer os.RemoveAll(actualFilesystemDir)

		paths := []string{
			path.Join(testDir, "upstream"),
			path.Join(testDir, "base"),
			path.Join(testDir, "overlays"),
		}

		tarGz := archiver.TarGz{
			Tar: &archiver.Tar{
				ImplicitTopLevelFolder: false,
			},
		}
		if err := tarGz.Archive(paths, path.Join(actualFilesystemDir, "archive.tar.gz")); err != nil {
			return errors.Wrap(err, "failed to create archive")
		}

		actual, err := ioutil.ReadFile(path.Join(actualFilesystemDir, "archive.tar.gz"))
		if err != nil {
			return errors.Wrap(err, "failed to read archive")
		}

		decodedExpected, err := base64.StdEncoding.DecodeString(pullTest.ExpectedFilesystem)
		if err != nil {
			return errors.Wrap(err, "failed to decode expected filesystem")
		}

		compareOptions := util.CompareOptions{
			IgnoreFilesInActual: []string{path.Join("upstream", "userdata", "license.yaml")},
		}

		ok, err := util.CompareTars(decodedExpected, actual, compareOptions)
		if err != nil {
			return errors.Wrap(err, "failed to compare tars")
		}

		if !ok {
			return errors.New("test failed, tar archives do not match")
		}

	}

	return nil
}
