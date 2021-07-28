package base

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/replicatedhq/kots/pkg/base"
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

		b, err := ioutil.ReadFile(testcaseFilepath)
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

		test.WantBase = baseFromDir(t, filepath.Join(path, "base"))

		// TODO: helm bases

		tests = append(tests, test)
	}
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// This disables the need for a Kubernetes cluster when running unit tests.
			template.TestingDisableKurlValues = true
			defer func() { template.TestingDisableKurlValues = false }()

			gotBase, _, err := base.RenderUpstream(&tt.Upstream, &tt.RenderOptions)
			require.NoError(t, err)

			if !assert.IsEqual(tt.WantBase, gotBase) {
				t.Log(diffJSON(gotBase, tt.WantBase))
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

			// TODO: gotHelmBases
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
	renderOptions.Log = logger.NewCLILogger()
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

		b, err := ioutil.ReadFile(path)
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

func baseFromDir(t *testing.T, root string) base.Base {
	b := base.Base{Bases: []base.Base{}}

	b.Files = baseFilesFromDir(t, root)

	return b
}

func baseFilesFromDir(t *testing.T, root string) []base.BaseFile {
	files := []base.BaseFile{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err, path)

		if info.IsDir() {
			return nil
		}

		b, err := ioutil.ReadFile(path)
		require.NoError(t, err, path)

		relPath, err := filepath.Rel(root, path)
		require.NoError(t, err, path)

		files = append(files, base.BaseFile{
			Path:    relPath,
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
