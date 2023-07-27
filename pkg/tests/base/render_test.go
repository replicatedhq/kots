package base

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/stretchr/testify/require"
	"gopkg.in/go-playground/assert.v1"
)

type TestCaseSpec struct {
	Name          string
	Upstream      upstreamtypes.Upstream
	RenderOptions base.RenderOptions
}

type testCase struct {
	Name          string
	Upstream      upstreamtypes.Upstream
	RenderOptions base.RenderOptions
	WantBase      base.Base
	WantHelmBase  base.Base
	WantKotsKinds *kotsutil.KotsKinds
}

func TestRenderUpstream(t *testing.T) {
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
			Name:          spec.Name,
			Upstream:      upstreamFromTestCaseSpec(spec),
			RenderOptions: renderOptionsFromTestCaseSpec(spec),
		}

		test.Upstream.Files = upstreamFilesFromDir(t, filepath.Join(path, "upstream"))

		test.WantBase = baseFromDir(t, filepath.Join(path, "base"), false)

		test.WantKotsKinds, err = kotsutil.LoadKotsKindsFromPath(filepath.Join(path, "kotsKinds"))
		require.NoError(t, err, "kotsKinds")

		chartsPath := filepath.Join(path, "base", "charts")
		if _, err := os.Stat(chartsPath); err == nil {
			charts, err := os.ReadDir(chartsPath)
			require.NoError(t, err)
			for _, chart := range charts {
				chartBase := baseFromDir(t, filepath.Join(chartsPath, chart.Name()), true)
				chartBase.Path = filepath.Join("charts", chart.Name())
				chartBase.ErrorFiles = []base.BaseFile{}
				test.WantBase.Bases = append(test.WantBase.Bases, chartBase)
			}
		}

		helmpath := filepath.Join(path, "basehelm")
		useHelmInstall := true
		if _, err := os.Stat(helmpath); err == nil {
			test.WantHelmBase = baseFromDir(t, filepath.Join(path, "basehelm"), useHelmInstall)
		}
		// Look for the native helm way of rendering
		for index, path := range test.WantHelmBase.Files {
			test.WantHelmBase.Files[index].Path = fmt.Sprintf("templates/%s", path.Path)
		}
		tests = append(tests, test)
	}
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// This disables the need for a Kubernetes cluster when running unit tests.
			template.TestingDisableKurlValues = true
			defer func() { template.TestingDisableKurlValues = false }()

			gotBase, gotHelmBases, gotKotsKindsFiles, err := base.RenderUpstream(&tt.Upstream, &tt.RenderOptions)
			require.NoError(t, err)

			if len(tt.WantBase.Files) > 0 {
				if !assert.IsEqual(tt.WantBase, gotBase) {
					t.Log(diffJSON(gotBase, tt.WantBase))
					t.FailNow()
				}

				if !assert.IsEqual(tt.WantBase.Files, gotBase.Files) {
					for idx := range tt.WantBase.Files {
						if len(gotBase.Files) > idx && gotBase.Files[idx].Path == tt.WantBase.Files[idx].Path {
							t.Log("FILE", tt.WantBase.Files[idx].Path)
							t.Log(diffString(string(gotBase.Files[idx].Content), string(tt.WantBase.Files[idx].Content)))
						}
					}
					t.FailNow()
				}
			}

			gotKotsKinds, err := kotsutil.KotsKindsFromMap(gotKotsKindsFiles)
			require.NoError(t, err, "kots kinds from map")

			if tt.WantKotsKinds.V1Beta1HelmCharts != nil && gotKotsKinds.V1Beta1HelmCharts != nil {
				require.ElementsMatch(t, tt.WantKotsKinds.V1Beta1HelmCharts.Items, gotKotsKinds.V1Beta1HelmCharts.Items)
				tt.WantKotsKinds.V1Beta1HelmCharts.Items = nil
				gotKotsKinds.V1Beta1HelmCharts.Items = nil
			}

			if tt.WantKotsKinds.V1Beta2HelmCharts != nil && gotKotsKinds.V1Beta2HelmCharts != nil {
				require.ElementsMatch(t, tt.WantKotsKinds.V1Beta2HelmCharts.Items, gotKotsKinds.V1Beta2HelmCharts.Items)
				tt.WantKotsKinds.V1Beta2HelmCharts.Items = nil
				gotKotsKinds.V1Beta2HelmCharts.Items = nil
			}

			require.Equal(t, tt.WantKotsKinds, gotKotsKinds)

			// TODO: Need to test upstream with multiple Helm charts.
			// HACK: Also right now "no files" in WantHelmBase implies test does not include any charts.
			if len(tt.WantHelmBase.Files) == 0 {
				return
			}

			if !assert.IsEqual(1, len(gotHelmBases)) {
				t.FailNow()
			}

			gotHelmBase := gotHelmBases[0] // TODO: add more helm charts

			helmTestFailed := false
			for _, wantFile := range tt.WantHelmBase.Files {
				contentFound := false
				for _, gotFile := range gotHelmBase.Files {
					if string(wantFile.Content) == string(gotFile.Content) {
						contentFound = true
						break
					}
				}
				if !assert.IsEqual(true, contentFound) {
					helmTestFailed = true
					t.Log("file content not found", wantFile.Content)
				}

				pathFound := false
				for _, gotFile := range gotHelmBase.Files {
					if wantFile.Path == gotFile.Path {
						pathFound = true
						break
					}
				}
				if !assert.IsEqual(true, pathFound) {
					helmTestFailed = true
					t.Log("file name not found", wantFile.Path)
				}
			}
			if helmTestFailed {
				t.FailNow()
			}
		})
	}
}

func upstreamFromTestCaseSpec(spec TestCaseSpec) upstreamtypes.Upstream {
	upstream := spec.Upstream
	if upstream.Type == "" {
		upstream.Type = "replicated"
	}
	if upstream.Name == "" {
		upstream.Name = "My App"
	}
	if upstream.UpdateCursor == "" {
		upstream.UpdateCursor = "123"
	}
	if upstream.ChannelID == "" {
		upstream.ChannelID = "1vusIYZLAVxMG6q760OJmRKj5i5"
	}
	if upstream.ChannelName == "" {
		upstream.ChannelName = "My Channel"
	}
	if upstream.VersionLabel == "" {
		upstream.VersionLabel = "1.0.1"
	}
	if upstream.ReleaseNotes == "" {
		upstream.ReleaseNotes = "Release 1.0.1"
	}
	if upstream.ReleasedAt == nil {
		t, err := time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
		if err != nil {
			panic(err)
		}
		upstream.ReleasedAt = &t
	}
	if upstream.EncryptionKey == "" {
		upstream.EncryptionKey = "4FxTKorpUz/cbjJnUcTBAnJrSWlUqhhiVFS1Nhime+xQP9Dl"
	}
	return upstream
}

func renderOptionsFromTestCaseSpec(spec TestCaseSpec) base.RenderOptions {
	renderOptions := spec.RenderOptions
	renderOptions.Log = logger.NewCLILogger(os.Stdout)
	renderOptions.Log.Silence()
	renderOptions.ExcludeKotsKinds = true
	renderOptions.SplitMultiDocYAML = true
	if renderOptions.AppSlug == "" {
		renderOptions.AppSlug = "my-app"
	}
	return renderOptions
}

func upstreamFilesFromDir(t *testing.T, root string) []upstreamtypes.UpstreamFile {
	files := []upstreamtypes.UpstreamFile{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err, path)

		if info.IsDir() {
			return nil
		}

		b, err := os.ReadFile(path)
		require.NoError(t, err, path)

		relPath, err := filepath.Rel(root, path)
		require.NoError(t, err, path)

		files = append(files, upstreamtypes.UpstreamFile{
			Path:    relPath,
			Content: b,
		})

		return nil
	})
	require.NoError(t, err)
	return files
}

func baseFromDir(t *testing.T, root string, isHelm bool) base.Base {
	b := base.Base{
		Bases: []base.Base{},
	}

	b.Files = baseFilesFromDir(t, root, isHelm)

	if !isHelm {
		return b
	}

	additionalFiles := []string{"values.yaml", "Chart.yaml"}
	for _, path := range additionalFiles {
		additionalFile := filepath.Join(root, path)
		if _, err := os.Stat(additionalFile); err == nil {
			data, err := os.ReadFile(additionalFile)
			require.NoError(t, err, additionalFile)

			b.AdditionalFiles = append(b.AdditionalFiles, base.BaseFile{
				Path:    path,
				Content: data,
			})
		}
	}

	return b
}

func baseFilesFromDir(t *testing.T, root string, isHelm bool) []base.BaseFile {
	files := []base.BaseFile{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err, path)

		if info.IsDir() {
			return nil
		}

		if isHelm && (info.Name() == "Chart.yaml" || info.Name() == "values.yaml") {
			// These files go into AdditonalFiles
			return nil
		}

		chartsPrefix := root + string(filepath.Separator) + "charts" + string(filepath.Separator)
		if !isHelm && strings.HasPrefix(path, chartsPrefix) {
			// Classic style charts are not Files. They become Bases.
			return nil
		}

		b, err := os.ReadFile(path)
		require.NoError(t, err, path)
		var fPath string
		fPath, err = filepath.Rel(root, path)
		require.NoError(t, err, fPath)
		if isHelm {
			fPath = filepath.Base(path)
		}

		files = append(files, base.BaseFile{
			Path:    fPath,
			Content: b,
		})

		return nil
	})
	require.NoError(t, err)
	return files
}

func diffJSON(got, want interface{}) string {
	a, _ := json.MarshalIndent(got, "", "  ")
	b, _ := json.MarshalIndent(want, "", "  ")
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(a)),
		B:        difflib.SplitLines(string(b)),
		FromFile: "Got",
		ToFile:   "Want",
		Context:  1,
	}
	diffStr, _ := difflib.GetUnifiedDiffString(diff)
	return fmt.Sprintf("got:\n%s \n\nwant:\n%s \n\ndiff:\n%s", got, want, diffStr)
}

func diffString(got, want string) string {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(got),
		B:        difflib.SplitLines(want),
		FromFile: "Got",
		ToFile:   "Want",
		Context:  1,
	}
	diffStr, _ := difflib.GetUnifiedDiffString(diff)
	return fmt.Sprintf("got:\n%s \n\nwant:\n%s \n\ndiff:\n%s", got, want, diffStr)
}
