package preflight

import (
	"path/filepath"
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/preflight/types"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/stretchr/testify/require"
)

func TestUnsupportedNoticeNeedsNoCluster(t *testing.T) {
	t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "missing-config"))
	kinds, err := kotsutil.KotsKindsFromMap(map[string][]byte{"beta3.yaml": kotsutil.UnsupportedTroubleshootWarning(), "bundle.yaml": kotsutil.UnsupportedTroubleshootWarning()})
	require.NoError(t, err)
	require.True(t, kinds.HasPreflights())
	require.Empty(t, kinds.Preflight.Spec.Collectors)
	require.Empty(t, kinds.Preflight.Spec.Analyzers)
	var saved *types.PreflightResults
	save := func(results *types.PreflightResults) error { saved = results; return nil }
	result, err := Execute(kinds.Preflight, false, nil, save, kinds.HasUnsupportedTroubleshoot)
	require.NoError(t, err)
	require.Same(t, result, saved)
	require.Empty(t, result.Results)
	require.Equal(t, kotsutil.UnsupportedTroubleshootMessage, result.Notice)
	require.Contains(t, GetPreflightCommand("app", kinds.HasUnsupportedTroubleshoot)[0], result.Notice)

	// Real checks must still enter collection instead of being bypassed.
	mixed := kinds.Preflight.DeepCopy()
	mixed.Spec.Analyzers = []*troubleshootv1beta2.Analyze{{ClusterVersion: &troubleshootv1beta2.ClusterVersion{}}}
	for _, unsupported := range []bool{false, true} {
		_, err = Execute(mixed, false, nil, save, unsupported)
		require.Error(t, err, "real checks require the missing cluster configuration")
		require.NotEmpty(t, saved.Errors)
		require.Empty(t, saved.Results, "a notice must not count as completed analysis")
		require.Equal(t, unsupported, saved.Notice != "")
	}
}
