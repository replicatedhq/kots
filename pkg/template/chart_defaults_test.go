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
	builder, _, err := NewBuilder(BuilderOptions{})
	require.NoError(t, err)
	chartBuilder, err := builder.WithHelmChart(parent)
	require.NoError(t, err)
	tests := []struct{ name, path, want string }{
		{"parent image", ".Values.image", "parent"},
		{"dependency image", ".Values.aliased.image", "child"},
		{"parent override", ".Values.aliased.replicas", "3"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := chartBuilder.String(`repl{{ HelmValue "` + test.path + `" }}`)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}

	_, err = builder.WithHelmChart([]byte("broken archive"))
	require.Error(t, err)
	_, err = builder.WithHelmChart(testChartArchive(t, "broken", "x: [", nil))
	require.Error(t, err)
}

func TestMalformedChartMetadataHasContext(t *testing.T) {
	builder, _, err := NewBuilder(BuilderOptions{})
	require.NoError(t, err)
	_, err = builder.WithHelmChart(testChartArchive(t, "broken", "", map[string][]byte{"Chart.yaml": []byte("name: [")}))
	require.ErrorContains(t, err, "parse Chart.yaml")
	_, err = builder.WithHelmChart(testChartArchive(t, "parent", "", map[string][]byte{"charts/child/Chart.yaml": []byte("name: [")}))
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
	builder, _, err := NewBuilder(BuilderOptions{})
	require.NoError(t, err)
	chartBuilder, err := builder.WithHelmChart(parent)
	require.NoError(t, err)
	got, err := chartBuilder.String(`repl{{ HelmValue ".Values.child.replicas" }}`)
	require.NoError(t, err)
	require.Equal(t, "7", got)
}

func TestHelmValuePathSyntaxAndArchiveLayouts(t *testing.T) {
	for _, prefix := range []string{"", "chart/", "./chart/"} {
		t.Run(prefix, func(t *testing.T) {
			var data bytes.Buffer
			gz := gzip.NewWriter(&data)
			tr := tar.NewWriter(gz)
			for name, content := range map[string]string{"Chart.yaml": "name: chart\n", "values.yaml": "log.level: debug\nlog:\n  level: nested\nstar*: literal\nimage[: bracket-key\nValuesFoo: literal-values-key\nFoo: wrong-key\nValues:\n  nested: literal-values-map\n"} {
				require.NoError(t, tr.WriteHeader(&tar.Header{Name: prefix + name, Size: int64(len(content)), Mode: 0644}))
				_, err := tr.Write([]byte(content))
				require.NoError(t, err)
			}
			require.NoError(t, tr.Close())
			require.NoError(t, gz.Close())
			b, _, err := NewBuilder(BuilderOptions{})
			require.NoError(t, err)
			b, err = b.WithHelmChart(data.Bytes())
			require.NoError(t, err)
			tests := []struct{ name, expression, want string }{
				{"nested key", `HelmValue ".Values.log.level"`, "nested"},
				{"literal dotted key", "HelmValue `" + `.Values.log\.level` + "`", "debug"},
				{"literal wildcard key", "HelmValue `" + `.Values.star\*` + "`", "literal"},
				{"missing key", `HelmValue ".Values.missing"`, ""},
				{"Values prefix is not a segment", `HelmValue ".ValuesFoo"`, "literal-values-key"},
				{"Values-prefixed key with root", `HelmValue ".Values.ValuesFoo"`, "literal-values-key"},
				{"Values map under root", `HelmValue ".Values.Values.nested"`, "literal-values-map"},
				{"bare path", `HelmValue "ValuesFoo"`, "literal-values-key"},
				{"exact Values root", `HelmValue ".Values"`, ""},
				{"literal bracket key", `HelmValue ".Values.image["`, "bracket-key"},
				{"unknown bracket key", `HelmValue ".Values.absent["`, ""},
			}
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					got, err := b.String("repl{{ " + test.expression + " }}")
					require.NoError(t, err)
					require.Equal(t, test.want, got)
				})
			}

		})
	}
}

func TestDependencyVersionsSelectMatchingDefaults(t *testing.T) {
	first := testChartArchive(t, "child", "value: first\n", nil)
	second := testChartArchive(t, "child", "value: second\n", map[string][]byte{"Chart.yaml": []byte("name: child\nversion: 2.0.0\n")})
	dependencies := []byte("name: parent\ndependencies:\n- name: child\n  alias: one\n  version: 1.0.0\n- name: child\n  alias: two\n  version: 2.0.0\n")
	parent := testChartArchive(t, "parent", "one:\n  override: retained\n", map[string][]byte{"Chart.yaml": dependencies, "charts/child-1.0.0.tgz": first, "charts/child-2.0.0.tgz": second})
	for i := 0; i < 10; i++ {
		b, _, err := NewBuilder(BuilderOptions{})
		require.NoError(t, err)
		b, err = b.WithHelmChart(parent)
		require.NoError(t, err)
		tests := []struct{ name, path, want string }{
			{"version one", "one.value", "first"},
			{"version two", "two.value", "second"},
			{"override retained", "one.override", "retained"},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				got, err := b.String(`repl{{ HelmValue "` + test.path + `" }}`)
				require.NoError(t, err)
				require.Equal(t, test.want, got)
			})
		}

	}
}

func TestAmbiguousDependencyDefaultsFailDeterministically(t *testing.T) {
	child := testChartArchive(t, "child", "value: child\n", nil)
	parent := testChartArchive(t, "parent", "", map[string][]byte{"charts/a.tgz": child, "charts/b.tgz": child})
	for i := 0; i < 10; i++ {
		b, _, err := NewBuilder(BuilderOptions{})
		require.NoError(t, err)
		_, err = b.WithHelmChart(parent)
		require.ErrorContains(t, err, "ambiguous dependency child: both a.tgz and b.tgz supply defaults")
	}
}

func TestHelmValueScalarMappingKeys(t *testing.T) {
	builder, _, err := NewBuilder(BuilderOptions{})
	require.NoError(t, err)
	builder, err = builder.WithHelmChart(testChartArchive(t, "keys", "large: 9007199254740993\n1: root\nnested:\n  2: number\n  true: boolean\n", nil))
	require.NoError(t, err)
	tests := []struct{ name, path, want string }{
		{"large integer", ".Values.large", "9007199254740993"},
		{"root numeric key", ".Values.1", "root"},
		{"nested numeric key", ".Values.nested.2", "number"},
		{"nested boolean key", ".Values.nested.true", "boolean"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := builder.String(`repl{{ HelmValue "` + test.path + `" }}`)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}
