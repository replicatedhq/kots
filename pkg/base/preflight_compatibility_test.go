package base

import (
	"strings"
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/stretchr/testify/require"
)

func TestTemplatedBeta3IsRetainedButNotExecuted(t *testing.T) {
	content := "apiVersion: troubleshoot.sh/v1beta3\r\nkind: SupportBundle\r\nspec:\r\n{{ if .Values.enabled }}\r\n  collectors: []\r\n{{ end }}\r\n"
	legacy := "apiVersion: troubleshoot.sh/v1beta2\nkind: Preflight\nspec:\n  analyzers:\n    - clusterVersion:\n        checkName: 'repl{{ printf \"KOTS check\" }}'\n"
	u := &upstreamtypes.Upstream{Type: "replicated", Files: []upstreamtypes.UpstreamFile{
		{Path: "legacy.yaml", Content: []byte(legacy)}, {Path: "beta3.yaml", Content: []byte(content)},
	}}
	opts := &RenderOptions{Namespace: "customer-app", ExcludeKotsKinds: true, Log: logger.NewCLILogger(nil)}
	files, err := RenderKotsKinds(u, opts)
	require.NoError(t, err)
	kinds, err := kotsutil.KotsKindsFromMap(files)
	require.NoError(t, err)
	require.Nil(t, kinds.HostPreflight)
	require.Nil(t, kinds.SupportBundle)
	require.Len(t, kinds.Preflight.Spec.Analyzers, 1)
	require.Equal(t, "KOTS check", kinds.Preflight.Spec.Analyzers[0].ClusterVersion.CheckName)
	require.True(t, kinds.HasUnsupportedTroubleshoot)
	b, _, err := RenderUpstream(u, opts, kinds)
	require.NoError(t, err)
	require.Empty(t, b.ListErrorFiles())
	require.Equal(t, content, string(u.Files[1].Content))
	_, err = kotsutil.LoadPreflightFromContents([]byte(content))
	require.ErrorContains(t, err, "KOTS does not support")
	_, err = kotsutil.LoadSupportBundleFromContents([]byte(content))
	require.ErrorContains(t, err, "KOTS does not support")
}

func TestPreflightReleaseDetection(t *testing.T) {
	legacy := "apiVersion: troubleshoot.sh/v1beta2\nkind: Preflight\nmetadata:\n  name: supported-checks\n  labels:\n    vendor: retained\nspec:\n  uri: https://example.com/checks.yaml\n  analyzers:\n    - clusterVersion: {}\n"
	beta3 := "apiVersion: troubleshoot.sh/v1beta3\nkind: Preflight\nspec:\n  collectors:\n    - runPod:\n        podSpec:\n          containers:\n            - resources:\n                requests:\n                  memory: '{{ .Values.memory }}'\n"
	excluded := "apiVersion: troubleshoot.sh/v1beta2\nkind: Preflight\nspec:\n  analyzers:\n    - clusterVersion:\n        exclude: true\n"
	for _, tc := range []struct {
		name             string
		docs             []string
		runnable, notice bool
	}{
		{"beta2 only", []string{legacy}, true, false},
		{"beta2 then beta3", []string{legacy, beta3}, true, true},
		{"beta3 then beta2", []string{beta3, legacy}, true, true},
		{"beta3 only", []string{beta3}, false, true},
		{"no preflights", nil, false, false},
		{"excluded beta2 only", []string{excluded}, false, false},
		{"excluded beta2 and beta3", []string{excluded, beta3}, false, true},
		{"multiple beta3", []string{beta3, beta3}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := &upstreamtypes.Upstream{Type: "replicated", Files: []upstreamtypes.UpstreamFile{{Path: "preflights.yaml", Content: []byte(strings.Join(tc.docs, "\n---\n"))}}}
			files, err := RenderKotsKinds(u, &RenderOptions{Namespace: "customer-app", Log: logger.NewCLILogger(nil)})
			require.NoError(t, err)
			kinds, err := kotsutil.KotsKindsFromMap(files)
			require.NoError(t, err)
			require.Equal(t, tc.runnable, kotsutil.HasRunnablePreflights(kinds.Preflight))
			require.Equal(t, tc.notice, kinds.HasUnsupportedTroubleshoot)
			require.Equal(t, tc.runnable || tc.notice, kinds.HasPreflights())
			if tc.runnable {
				require.Equal(t, "https://example.com/checks.yaml", kinds.Preflight.Spec.Uri)
				require.Equal(t, "supported-checks", kinds.Preflight.Name)
				require.Equal(t, "retained", kinds.Preflight.Labels["vendor"])
			}
			// GetApp reads this merged, persisted document rather than the archive.
			stored, err := kinds.Marshal("troubleshoot.sh", "v1beta2", "Preflight")
			require.NoError(t, err)
			if stored != "" {
				reloaded, err := kotsutil.KotsKindsFromMap(map[string][]byte{"preflight.yaml": []byte(stored)})
				require.NoError(t, err)
				require.Equal(t, tc.notice, reloaded.HasUnsupportedTroubleshoot)
				require.Equal(t, kinds.HasPreflights(), reloaded.HasPreflights())
			}
		})
	}
}

func TestConfigMapWithEmbeddedBeta3IsDeployed(t *testing.T) {
	content := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: embedded-checks\ndata:\n  preflight.yaml: |\n    apiVersion: troubleshoot.sh/v1beta3\n    kind: Preflight\nrepl{{ if true }}\n  enabled: 'true'\nrepl{{ end }}\n"
	u := &upstreamtypes.Upstream{Type: "replicated", Files: []upstreamtypes.UpstreamFile{{Path: "configmap.yaml", Content: []byte(content)}}}
	opts := &RenderOptions{Namespace: "customer-app", ExcludeKotsKinds: true, Log: logger.NewCLILogger(nil)}
	files, err := RenderKotsKinds(u, opts)
	require.NoError(t, err)
	kinds, err := kotsutil.KotsKindsFromMap(files)
	require.NoError(t, err)
	require.False(t, kinds.HasUnsupportedTroubleshoot)
	b, _, err := RenderUpstream(u, opts, kinds)
	require.NoError(t, err)
	require.Empty(t, b.ListErrorFiles())
	require.Len(t, b.Files, 1)
	require.Contains(t, string(b.Files[0].Content), "troubleshoot.sh/v1beta3")
}
