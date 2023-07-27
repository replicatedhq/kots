package pull

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	envsubst "github.com/drone/envsubst/v2"
	"github.com/ghodss/yaml"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/pull"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/assert"
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

		b, err := os.ReadFile(testcaseFilepath)
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

					if info.IsDir() && info.Name() == "kotsKinds" {
						return filepath.SkipDir
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

					if info.IsDir() && info.Name() == "kotsKinds" {
						return filepath.SkipDir
					}

					if info.IsDir() {
						return nil
					}

					// exclude installation.yaml as it has a randomly generated encryptionKey
					if strings.HasSuffix(path, "upstream/userdata/installation.yaml") ||
						strings.HasSuffix(path, ".DS_Store") {
						return nil
					}

					contents, err := os.ReadFile(path)
					if err != nil {
						return err
					}

					wantPath := strings.Replace(path, "results", "wantResults", 1)

					wantContents, err := os.ReadFile(wantPath)
					if err != nil {
						fmt.Printf("unable to open file %s\n", wantPath)
					}
					require.NoError(t, err, wantPath)

					contentsString := string(contents)
					wantContentsString := string(wantContents)

					if ext := filepath.Ext(wantPath); ext == ".yaml" || ext == ".yml" {
						wantContentsString, err = envsubst.Eval(wantContentsString, util.TestGetenv)
						require.NoError(t, err, wantPath)
					}

					assert.Equal(t, wantContentsString, contentsString, wantPath)
					return nil
				})

			require.NoError(t, err)

			// Check that kots kinds match.
			installationPath := ""
			err = filepath.Walk(filepath.Join(tt.PullOptions.RootDir, "kotsKinds"),
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if info.IsDir() {
						return nil
					}

					// Skip installation.yaml because known images order will differ on every generation.
					if strings.HasSuffix(path, "installation.yaml") {
						installationPath = path
						return nil
					}

					rawContents, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					require.NoError(t, err, path)
					contents := util.ConvertToSingleDocs(rawContents)

					kotsKinds := []*kotsutil.KotsKinds{}
					for _, content := range contents {
						elem, err := kotsutil.KotsKindsFromMap(map[string][]byte{path: content})
						require.NoError(t, err, path)
						kotsKinds = append(kotsKinds, elem)
					}

					wantPath := strings.Replace(path, "results", "wantResults", 1)

					rawWantContents, err := os.ReadFile(wantPath)
					if err != nil {
						fmt.Printf("unable to open file %s\n", wantPath)
					}
					require.NoError(t, err, wantPath)
					wantContents := util.ConvertToSingleDocs(rawWantContents)

					wantKotsKinds := []*kotsutil.KotsKinds{}
					for _, content := range wantContents {
						elem, err := kotsutil.KotsKindsFromMap(map[string][]byte{path: content})
						require.NoError(t, err, path)
						wantKotsKinds = append(wantKotsKinds, elem)
					}

					require.ElementsMatch(t, wantKotsKinds, kotsKinds)
					return nil
				})

			require.NoError(t, err)

			wantInstallationPath := strings.Replace(installationPath, "results", "wantResults", 1)

			installationContents, err := os.ReadFile(installationPath)
			require.NoError(t, err)

			wantInstallationContents, err := os.ReadFile(wantInstallationPath)
			require.NoError(t, err)

			installation := kotsv1beta1.Installation{}
			err = yaml.Unmarshal(installationContents, &installation)
			require.NoError(t, err)

			wantInstallation := kotsv1beta1.Installation{}
			err = yaml.Unmarshal(wantInstallationContents, &wantInstallation)
			require.NoError(t, err)

			require.ElementsMatch(t, wantInstallation.Spec.KnownImages, installation.Spec.KnownImages)
			wantInstallation.Spec.KnownImages = nil
			installation.Spec.KnownImages = nil
			require.Equal(t, wantInstallation, installation)
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
