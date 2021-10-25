package pull

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/replicatedhq/kots/pkg/pull"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/stretchr/testify/require"
)

type TestCaseSpec struct {
	Name        string
	PullOptions pull.PullOptions
	ResultsDir  string
}

type testCase struct {
	Name        string
	UpstreamURI string
	PullOptions pull.PullOptions
	Upstream    upstreamtypes.Upstream
}

func TestKotsPull(t *testing.T) {
	tests := []testCase{}

	root := "cases"

	entries, err := os.ReadDir(root)
	require.NoError(t, err)

	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())

		if !entry.IsDir() {
			t.Logf("unexpected file %s", path)
			continue
		}

		upstreamURI := filepath.Join(path, "upstream")
		_, err = os.Stat(upstreamURI)
		if os.IsNotExist(err) {
			t.Logf("no upstream directory found in test directory %s", path)
			continue
		} else {
			require.NoError(t, err, path)
		}

		// prepend upstreamURI with replicated:// scheme
		upstream := pull.RewriteUpstream(upstreamURI)

		testcaseFilepath := filepath.Join(path, "testcase.yaml")
		_, err = os.Stat(testcaseFilepath)
		if os.IsNotExist(err) {
			t.Logf("no testcase.yaml found in directory %s", path)
			continue
		} else {
			require.NoError(t, err, path)
		}

		b, err := ioutil.ReadFile(testcaseFilepath)
		require.NoError(t, err, path)

		var spec TestCaseSpec
		err = yaml.Unmarshal(b, &spec)
		require.NoError(t, err, path)

		test := testCase{
			Name:        spec.Name,
			UpstreamURI: upstream,
			PullOptions: pullOptionsFromTestCaseSpec(spec),
		}

		tests = append(tests, test)
	}
	require.NoError(t, err)

	// ensure the tests are actually loaded
	if len(tests) == 0 {
		fmt.Printf("Kots Pull test cases not found")
		t.FailNow()
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// create the result directories and defer cleanup
			os.Mkdir(tt.PullOptions.RootDir, 0755)
			os.Mkdir(fmt.Sprintf("%s/replicated-kots-app", tt.PullOptions.RootDir), 0755)

			// Comment out this line to review the generated results.
			// Just be sure to delete the 'results' directory in each test manually when finished!
			defer func() { os.RemoveAll(tt.PullOptions.RootDir) }()

			fmt.Printf("running test %s\n", tt.Name)
			_, err := pull.Pull(tt.UpstreamURI, tt.PullOptions)
			require.NoError(t, err)

			wantResultsDir := strings.Replace(tt.PullOptions.RootDir, "results", "wantResults", 1)
			err = filepath.Walk(wantResultsDir,
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if info.IsDir() {
						return nil
					}

					resultPath := strings.Replace(path, "wantResults", "results", 1)

					if _, err = os.Stat(resultPath); os.IsNotExist(err) {
						fmt.Printf("expected file %s not found in results\n", resultPath)
						t.FailNow()
					}

					return nil
				})

			require.NoError(t, err)

			// compare result files to wanted files
			err = filepath.Walk(tt.PullOptions.RootDir,
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if info.IsDir() {
						return nil
					}

					// exclude installation.yaml as it has a randomly generated encryptionKey
					if strings.HasSuffix(path, "upstream/userdata/installation.yaml") {
						return nil
					}

					contents, err := ioutil.ReadFile(path)
					if err != nil {
						return err
					}

					wantPath := strings.Replace(path, "results", "wantResults", 1)

					wantContents, err := ioutil.ReadFile(wantPath)
					require.NoError(t, err)

					contentsString := string(contents)
					wantContentsString := string(wantContents)
					require.Equal(t, contentsString, wantContentsString)

					return nil
				})

			require.NoError(t, err)
		})
	}
}

func pullOptionsFromTestCaseSpec(spec TestCaseSpec) pull.PullOptions {
	pullOptions := spec.PullOptions
	pullOptions.ExcludeKotsKinds = true
	if pullOptions.AppSlug == "" {
		pullOptions.AppSlug = "my-app"
	}
	return pullOptions
}
