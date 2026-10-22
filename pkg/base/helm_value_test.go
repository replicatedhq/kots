package base

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/stretchr/testify/require"
)

func releaseChart(t *testing.T, name string, overrides ...map[string]string) upstreamtypes.UpstreamFile {
	t.Helper()
	files := map[string]string{"Chart.yaml": "apiVersion: v2\nname: " + name + "\nversion: 1.0.0\n", "values.yaml": "image: " + name + "\n"}
	for _, override := range overrides {
		for path, content := range override {
			files[path] = content
		}
	}
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	tr := tar.NewWriter(gz)
	for file, content := range files {
		require.NoError(t, tr.WriteHeader(&tar.Header{Name: name + "/" + file, Size: int64(len(content)), Mode: 0644}))
		_, err := tr.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tr.Close())
	require.NoError(t, gz.Close())
	return upstreamtypes.UpstreamFile{Path: name + ".tgz", Content: buffer.Bytes()}
}

func TestHelmValueUsesMatchingReleaseChart(t *testing.T) {
	files := []upstreamtypes.UpstreamFile{releaseChart(t, "first"), releaseChart(t, "second")}
	tests := []struct{ name, chartName string }{{"first chart", "first"}, {"second chart", "second"}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := []byte("apiVersion: kots.io/v1beta2\nkind: HelmChart\nspec:\n  chart: {name: " + test.chartName + ", chartVersion: 1.0.0}\n  values:\n    message: '{{repl/* repl{{ bad */}}'\n    image: 'repl{{ HelmValue \".Values.image\" }}'\n")
			input := append(append([]upstreamtypes.UpstreamFile{}, files...), upstreamtypes.UpstreamFile{Path: "chart.yaml", Content: document})
			rendered, err := RenderKotsKinds(&upstreamtypes.Upstream{Files: input}, &RenderOptions{Log: logger.NewCLILogger(io.Discard)})
			require.NoError(t, err)
			require.Contains(t, string(rendered["chart.yaml"]), "image: '"+test.chartName+"'")
		})
	}
	b, _, err := template.NewBuilder(template.BuilderOptions{})
	require.NoError(t, err)
	result, err := b.String(`repl{{ HelmValue ".Values.image" }}`)
	require.NoError(t, err)
	require.Empty(t, result)
}

func TestHelmValueArchiveResolutionOnlyForFunctionCalls(t *testing.T) {
	tests := []struct{ name, value, wantError string }{
		{name: "literal mention", value: "HelmValue"},
		{name: "first pass removes invalid second delimiter", value: `{{repl/* repl{{ bad */}}`},
		{name: "inactive call needs no archive", value: `repl{{ if false }}repl{{ HelmValue ".image" }}repl{{ end }}`},
		{name: "genuine invalid template still fails", value: `repl{{ NotAFunction }}`, wantError: "NotAFunction"},
		{name: "string argument", value: `repl{{ printf "%s" "HelmValue" }}`},
		{name: "template comment", value: `repl{{/* HelmValue */}}`},
		{name: "actual call", value: `repl{{ HelmValue ".image" }}`, wantError: "failed to resolve archive for chart missing 1.0.0"},
		{name: "alternate delimiter", value: `{{repl HelmValue ".image" }}`, wantError: "failed to resolve archive for chart missing 1.0.0"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := []byte("apiVersion: kots.io/v1beta2\nkind: HelmChart\nspec:\n  chart: {name: missing, chartVersion: 1.0.0}\n  values:\n    message: '" + test.value + "'\n")
			_, err := RenderKotsKinds(&upstreamtypes.Upstream{Files: []upstreamtypes.UpstreamFile{{Path: "chart.yaml", Content: document}}}, &RenderOptions{Log: logger.NewCLILogger(io.Discard)})
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				require.ErrorContains(t, err, "chart.yaml")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestHelmValueArchiveErrorIdentifiesOffendingFile(t *testing.T) {
	var corrupt bytes.Buffer
	gz := gzip.NewWriter(&corrupt)
	_, err := gz.Write([]byte("invalid tar payload"))
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	document := []byte(`apiVersion: kots.io/v1beta2
kind: HelmChart
spec:
  chart: {name: requested, chartVersion: 1.0.0}
  values:
    image: 'repl{{ HelmValue ".image" }}'
`)
	_, err = RenderKotsKinds(&upstreamtypes.Upstream{Files: []upstreamtypes.UpstreamFile{
		{Path: "other-chart.tgz", Content: corrupt.Bytes()},
		releaseChart(t, "requested"),
		{Path: "requested-chart.yaml", Content: document},
	}}, &RenderOptions{Log: logger.NewCLILogger(io.Discard)})
	require.ErrorContains(t, err, "failed to render file requested-chart.yaml")
	require.ErrorContains(t, err, "failed to resolve archive for chart requested 1.0.0")
	require.ErrorContains(t, err, "failed to read chart archive other-chart.tgz")
	require.ErrorContains(t, err, "unexpected EOF")
}

func TestRenderUpstreamPreservesMultiDocumentChartDiscovery(t *testing.T) {
	content := []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: ordinary
---
apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: previously-skipped
spec:
  chart:
    name: missing
    chartVersion: 1.0.0
`)
	base, helmBases, err := RenderUpstream(&upstreamtypes.Upstream{Type: "replicated", Files: []upstreamtypes.UpstreamFile{{Path: "mixed.yaml", Content: content}}}, &RenderOptions{ExcludeKotsKinds: true, Log: logger.NewCLILogger(io.Discard)}, &kotsutil.KotsKinds{})
	require.NoError(t, err)
	require.Empty(t, helmBases)
	require.Empty(t, base.Bases)
}

func TestHelmValueCannotResolveItsOwnChartIdentity(t *testing.T) {
	tests := []struct{ name, chartName, version string }{
		{name: "chart name", chartName: `repl{{ HelmValue ".name" }}`, version: "1.0.0"},
		{name: "chart version", chartName: "first", version: `{{repl HelmValue ".version" }}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := []byte(fmt.Sprintf(`apiVersion: kots.io/v1beta2
kind: HelmChart
spec:
  chart:
    name: '%s'
    chartVersion: '%s'
  values:
    image: 'repl{{ HelmValue ".image" }}'
`, test.chartName, test.version))
			_, err := RenderKotsKinds(&upstreamtypes.Upstream{Files: []upstreamtypes.UpstreamFile{releaseChart(t, "first"), {Path: "chart.yaml", Content: document}}}, &RenderOptions{Log: logger.NewCLILogger(io.Discard)})
			require.ErrorContains(t, err, "chart.yaml")
			require.ErrorContains(t, err, "resolve HelmValue chart defaults")
			require.ErrorContains(t, err, "cannot resolve chart name or version before the chart archive is known")
		})
	}
}

func TestRenderKotsKindsPreservesOrdinaryHelmChartSiblings(t *testing.T) {
	ordinary := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: sibling\ndata:\n  value: 'repl{{ printf \"%s\" \"rendered\" }}'\n"
	chart := "apiVersion: kots.io/v1beta2\nkind: HelmChart\nspec:\n  chart: {name: first, chartVersion: 1.0.0}\n  values:\n    image: 'repl{{ HelmValue \".image\" }}'\n"
	for _, content := range []string{ordinary + "---\n" + chart, chart + "---\n" + ordinary + "---\n" + chart} {
		rendered, err := RenderKotsKinds(&upstreamtypes.Upstream{Files: []upstreamtypes.UpstreamFile{
			releaseChart(t, "first"),
			{Path: "mixed.yaml", Content: []byte(content)},
			{Path: "ordinary.yaml", Content: []byte(ordinary)},
		}}, &RenderOptions{Log: logger.NewCLILogger(io.Discard)})
		require.NoError(t, err)
		require.Contains(t, string(rendered["mixed.yaml"]), "value: 'rendered'")
		require.Equal(t, 1, strings.Count(string(rendered["mixed.yaml"]), "kind: ConfigMap"))
		require.Equal(t, strings.Count(content, "kind: HelmChart"), strings.Count(string(rendered["mixed.yaml"]), "image: 'first'"))
		require.NotContains(t, rendered, "ordinary.yaml")
	}
}

func TestRenderKotsKindsPreservesSiblingsThatFailGVKParsing(t *testing.T) {
	chart := "apiVersion: kots.io/v1beta2\nkind: HelmChart\nspec:\n  chart: {name: first, chartVersion: 1.0.0}\n"
	tests := []struct{ name, sibling, want string }{
		{"malformed YAML", "broken: [\nmessage: 'repl{{ printf \"%s\" \"rendered\" }}'\n", "broken: [\nmessage: 'rendered'"},
		{"non-string kind", "apiVersion: v1\nkind: [ConfigMap]\nmessage: 'repl{{ printf \"%s\" \"rendered\" }}'\n", "kind: [ConfigMap]\nmessage: 'rendered'"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, content := range []string{test.sibling + "---\n" + chart, chart + "---\n" + test.sibling} {
				rendered, err := RenderKotsKinds(&upstreamtypes.Upstream{Files: []upstreamtypes.UpstreamFile{
					{Path: "mixed.yaml", Content: []byte(content)},
					{Path: "ordinary.yaml", Content: []byte(test.sibling)},
				}}, &RenderOptions{Log: logger.NewCLILogger(io.Discard)})
				require.NoError(t, err)
				require.Contains(t, string(rendered["mixed.yaml"]), test.want)
				require.Equal(t, 1, strings.Count(string(rendered["mixed.yaml"]), "kind: HelmChart"))
				require.Equal(t, 1, strings.Count(string(rendered["mixed.yaml"]), "message: 'rendered'"))
				require.NotContains(t, rendered, "ordinary.yaml")
			}
		})
	}
}
