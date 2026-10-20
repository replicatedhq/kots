package base

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/replicatedhq/kots/pkg/template"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/stretchr/testify/require"
)

func TestHelmValueUsesMatchingReleaseChart(t *testing.T) {
	b, _, err := template.NewBuilder(template.BuilderOptions{})
	require.NoError(t, err)
	files := []upstreamtypes.UpstreamFile{}
	for _, name := range []string{"first", "second"} {
		var buffer bytes.Buffer
		gz := gzip.NewWriter(&buffer)
		tr := tar.NewWriter(gz)
		for file, content := range map[string]string{"Chart.yaml": "name: " + name + "\nversion: 1.0.0\n", "values.yaml": "image: " + name + "\n"} {
			require.NoError(t, tr.WriteHeader(&tar.Header{Name: name + "/" + file, Size: int64(len(content)), Mode: 0644}))
			_, err := tr.Write([]byte(content))
			require.NoError(t, err)
		}
		require.NoError(t, tr.Close())
		require.NoError(t, gz.Close())
		files = append(files, upstreamtypes.UpstreamFile{Path: name + ".tgz", Content: buffer.Bytes()})
	}
	for _, name := range []string{"first", "second"} {
		document := []byte("apiVersion: kots.io/v1beta2\nkind: HelmChart\nspec:\n  chart: {name: " + name + ", chartVersion: 1.0.0}\n  values:\n    image: 'repl{{ HelmValue \".Values.image\" }}'\n")
		chartBuilder, err := helmChartTemplateBuilder(document, files, b)
		require.NoError(t, err)
		result, err := chartBuilder.String(string(document))
		require.NoError(t, err)
		require.Contains(t, result, "image: '"+name+"'")
	}
	result, err := b.String(`repl{{ HelmValue ".Values.image" }}`)
	require.NoError(t, err)
	require.Empty(t, result)
}
