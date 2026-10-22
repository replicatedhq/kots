package base

import (
	"bytes"
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/stretchr/testify/require"
)

func TestMalformedBeta3ProducesWarning(t *testing.T) {
	u := &upstreamtypes.Upstream{Type: "replicated", Files: []upstreamtypes.UpstreamFile{{Path: "preflight.yaml", Content: []byte("apiVersion: troubleshoot.sh/v1beta3\nkind: Preflight\nspec:\n  collectors: [\n")}}}
	var logs bytes.Buffer
	opts := &RenderOptions{ExcludeKotsKinds: true, Log: logger.NewCLILogger(&logs)}
	files, err := RenderKotsKinds(u, opts)
	require.NoError(t, err)
	kinds, err := kotsutil.KotsKindsFromMap(files)
	require.NoError(t, err)
	b, _, err := RenderUpstream(u, opts, kinds)
	require.NoError(t, err)
	require.True(t, kinds.HasUnsupportedTroubleshoot)
	require.True(t, kinds.HasPreflights())
	require.Contains(t, logs.String(), "Skipping preflight.yaml: invalid v1beta3 Troubleshoot YAML")
	require.Empty(t, b.Files, "do not deploy the malformed preflight")
}
