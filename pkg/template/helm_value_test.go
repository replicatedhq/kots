package template

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHelmValueChartContext(t *testing.T) {
	b, _, err := NewBuilder(BuilderOptions{})
	require.NoError(t, err)
	child := testChartArchive(t, "child", "image: child\n", nil)
	parent := testChartArchive(t, "parent", "image: parent\n", map[string][]byte{"charts/child-1.0.0.tgz": child})
	withChart, err := b.WithHelmChart(parent)
	require.NoError(t, err)
	result, err := withChart.String(`repl{{ HelmValue ".Values.child.image" }}`)
	require.NoError(t, err)
	require.Equal(t, "child", result)
	_, err = b.WithHelmChart([]byte("broken"))
	require.Error(t, err)
}
