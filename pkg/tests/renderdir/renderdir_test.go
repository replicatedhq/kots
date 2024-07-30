package renderdir

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	envsubst "github.com/drone/envsubst/v2"
	"github.com/ghodss/yaml"
	"github.com/golang/mock/gomock"
	cp "github.com/otiai10/copy"
	"github.com/replicatedhq/kots/pkg/render"
	rendertypes "github.com/replicatedhq/kots/pkg/render/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestCaseSpec struct {
	Name             string
	RenderDirOptions rendertypes.RenderDirOptions
	ResultsDir       string
}

type testCase struct {
	Name             string
	RenderDirOptions rendertypes.RenderDirOptions
}

func TestKotsRenderDir(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Setenv("USE_MOCK_REPORTING", "1")
	defer os.Unsetenv("USE_MOCK_REPORTING")

	t.Setenv("KOTSADM_TARGET_NAMESPACE", "app-namespace")
	defer os.Unsetenv("KOTSADM_TARGET_NAMESPACE")

	tests := []testCase{}
	root := "cases"

	entries, err := os.ReadDir(root)
	require.NoError(t, err)

	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())

		if !entry.IsDir() {
			continue
		}

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
			Name:             spec.Name,
			RenderDirOptions: spec.RenderDirOptions,
		}
		test.RenderDirOptions.App.SelectedChannelID = "1vusIYZLAVxMG6q760OJmRKj5i5"
		tests = append(tests, test)
	}
	require.NoError(t, err)

	// ensure the tests are actually loaded
	if len(tests) == 0 {
		fmt.Printf("Kots RenderDir test cases not found")
		t.FailNow()
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			testDir := filepath.Dir(tt.RenderDirOptions.ArchiveDir)

			// copy archiveDir to preserve original test data
			resultsDir := filepath.Join(testDir, "results")
			os.Mkdir(resultsDir, 0755)
			err := cp.Copy(tt.RenderDirOptions.ArchiveDir, resultsDir)
			require.NoError(t, err)
			tt.RenderDirOptions.ArchiveDir = resultsDir

			// Comment out this line to review the generated results.
			// Just be sure to delete the 'results' directory in each test manually when finished!
			defer func() { os.RemoveAll(resultsDir) }()

			fmt.Printf("running test %s\n", tt.Name)
			err = render.RenderDir(tt.RenderDirOptions)
			require.NoError(t, err)

			wantResultsDir := filepath.Join(testDir, "wantResults")
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
			err = filepath.Walk(resultsDir,
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

					if strings.HasSuffix(wantPath, "pullsecrets.yaml") {
						// pull secret patches are not generated in a deterministic order
						gotPullSecrets := strings.Split(contentsString, "---\n")
						wantPullSecrets := strings.Split(wantContentsString, "---\n")
						require.ElementsMatch(t, wantPullSecrets, gotPullSecrets)
						return nil
					}

					assert.Equal(t, wantContentsString, contentsString, wantPath)
					return nil
				})

			require.NoError(t, err)

			// Check that kots kinds match.
			installationPath := ""
			err = filepath.Walk(filepath.Join(resultsDir, "kotsKinds"),
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

					wantPath := strings.Replace(path, "results", "wantResults", 1)
					rawWantContents, err := os.ReadFile(wantPath)
					if err != nil {
						fmt.Printf("unable to open file %s\n", wantPath)
					}
					require.NoError(t, err, wantPath)

					require.Equal(t, string(rawWantContents), string(rawContents), wantPath)
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
