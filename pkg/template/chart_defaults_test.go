package template

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/stretchr/testify/require"
)

func testChartArchive(t *testing.T, name, values string, extra map[string][]byte) []byte {
	t.Helper()
	files := map[string][]byte{"Chart.yaml": []byte("apiVersion: v2\nname: " + name + "\nversion: 1.0.0\n"), "values.yaml": []byte(values)}
	for key, data := range extra {
		files[key] = data
	}
	var data bytes.Buffer
	gz := gzip.NewWriter(&data)
	tr := tar.NewWriter(gz)
	for file, content := range files {
		require.NoError(t, tr.WriteHeader(&tar.Header{Name: name + "/" + file, Size: int64(len(content)), Mode: 0644}))
		_, err := tr.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, tr.Close())
	require.NoError(t, gz.Close())
	return data.Bytes()
}

func TestPackagedChartDefaults(t *testing.T) {
	child := testChartArchive(t, "child", "image: child\nreplicas: 1\n", nil)
	parent := testChartArchive(t, "parent", "image: parent\naliased:\n  replicas: 3\n", map[string][]byte{
		"Chart.yaml":             []byte("name: parent\ndependencies:\n- name: child\n  alias: aliased\n"),
		"charts/child-1.0.0.tgz": child,
	})
	values, err := chartDefaults(parent)
	require.NoError(t, err)
	require.Equal(t, "parent", values["image"])
	require.Equal(t, map[string]any{"image": "child", "replicas": 3}, values["aliased"])
	_, err = chartDefaults([]byte("broken archive"))
	require.Error(t, err)
	_, err = chartDefaults(testChartArchive(t, "broken", "x: [", nil))
	require.Error(t, err)
}

func TestMalformedChartMetadataHasContext(t *testing.T) {
	_, err := chartDefaults(testChartArchive(t, "broken", "", map[string][]byte{"Chart.yaml": []byte("name: [")}))
	require.ErrorContains(t, err, "parse Chart.yaml")
	_, err = chartDefaults(testChartArchive(t, "parent", "", map[string][]byte{"charts/child/Chart.yaml": []byte("name: [")}))
	require.ErrorContains(t, err, "parse dependency child Chart.yaml")
}

func TestLargeDependencyArchiveDefaults(t *testing.T) {
	// A valid dependency can exceed 32 MiB because of assets unrelated to defaults.
	var data bytes.Buffer
	gz, err := gzip.NewWriterLevel(&data, gzip.NoCompression)
	require.NoError(t, err)
	tr := tar.NewWriter(gz)
	for name, content := range map[string][]byte{
		"child/Chart.yaml":     []byte("name: child\n"),
		"child/values.yaml":    []byte("replicas: 7\n"),
		"child/assets/padding": make([]byte, 33<<20),
	} {
		require.NoError(t, tr.WriteHeader(&tar.Header{Name: name, Size: int64(len(content)), Mode: 0644}))
		_, err := tr.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, tr.Close())
	require.NoError(t, gz.Close())
	require.Greater(t, data.Len(), 32<<20)
	parent := testChartArchive(t, "parent", "", map[string][]byte{"charts/child.tgz": data.Bytes()})
	values, err := chartDefaults(parent)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"replicas": 7}, values["child"])
}
