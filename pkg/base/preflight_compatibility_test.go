package base

import (
	"strings"
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/stretchr/testify/require"
)

func TestPreflightReleaseDetection(t *testing.T) {
	legacy := "apiVersion: troubleshoot.sh/v1beta2\nkind: Preflight\nmetadata:\n  name: supported-checks\n  labels:\n    vendor: retained\nspec:\n  uri: https://example.com/checks.yaml\n  analyzers:\n    - clusterVersion:\n        checkName: 'repl{{ printf \"KOTS check\" }}'\n"
	beta3 := "apiVersion: troubleshoot.sh/v1beta3\nkind: Preflight\nspec:\n  collectors:\n    - runPod:\n        podSpec:\n          containers:\n            - resources:\n                requests:\n                  memory: '{{ .Values.memory }}'\n"
	bundle := "apiVersion: troubleshoot.sh/v1beta3\r\nkind: SupportBundle\r\nspec:\r\n{{ if .Values.enabled }}\r\n  collectors: []\r\n{{ end }}\r\n"
	excluded := "apiVersion: troubleshoot.sh/v1beta2\nkind: Preflight\nspec:\n  analyzers:\n    - clusterVersion:\n        exclude: true\n"
	embedded := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: embedded-checks\ndata:\n  preflight.yaml: |\n    apiVersion: troubleshoot.sh/v1beta3\n    kind: Preflight\nrepl{{ if true }}\n  enabled: 'true'\nrepl{{ end }}\n"
	for _, tc := range []struct {
		name             string
		docs             []string
		runnable, notice bool
	}{
		{"beta2 only", []string{legacy}, true, false},
		{"concrete beta3 only", []string{strings.ReplaceAll(legacy, "v1beta2", "v1beta3")}, true, false},
		{"beta2 and concrete beta3", []string{legacy, strings.ReplaceAll(legacy, "v1beta2", "v1beta3")}, true, false},
		{"beta2 then beta3", []string{legacy, beta3}, true, true},
		{"beta3 then beta2", []string{beta3, legacy}, true, true},
		{"beta3 only", []string{beta3}, false, true},
		{"no preflights", nil, false, false},
		{"excluded beta2 only", []string{excluded}, false, false},
		{"excluded beta2 and beta3", []string{excluded, beta3}, false, true},
		{"multiple beta3", []string{beta3, beta3}, false, true},
		{"KOTS checks and CRLF bundle template", []string{legacy, bundle}, true, true},
		{"embedded beta3", []string{embedded}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := &upstreamtypes.Upstream{Type: "replicated", Files: []upstreamtypes.UpstreamFile{{Path: "preflights.yaml", Content: []byte(strings.Join(tc.docs, "\n---\n"))}}}
			opts := &RenderOptions{Namespace: "customer-app", ExcludeKotsKinds: true, Log: logger.NewCLILogger(nil)}
			files, err := RenderKotsKinds(u, opts)
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
				require.Equal(t, "KOTS check", kinds.Preflight.Spec.Analyzers[0].ClusterVersion.CheckName)
			}
			base, _, err := RenderUpstream(u, opts, kinds)
			require.NoError(t, err)
			require.Empty(t, base.ListErrorFiles())
			if tc.name == "embedded beta3" {
				require.Len(t, base.Files, 1)
				require.Contains(t, string(base.Files[0].Content), "troubleshoot.sh/v1beta3")
			}
			require.Equal(t, strings.Join(tc.docs, "\n---\n"), string(u.Files[0].Content))
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

func TestConcreteBeta3KindsAndMalformedSpecs(t *testing.T) {
	for _, version := range []string{"v1beta2", "v1beta3"} {
		t.Run(version, func(t *testing.T) {
			bundle := []byte("apiVersion: troubleshoot.sh/" + version + "\nkind: SupportBundle\nspec:\n  collectors:\n    - clusterInfo: {}\n")
			loaded, err := kotsutil.LoadSupportBundleFromContents(bundle)
			require.NoError(t, err)
			require.Len(t, loaded.Spec.Collectors, 1)
			kinds, err := kotsutil.KotsKindsFromMap(map[string][]byte{"bundle.yaml": bundle,
				"host.yaml": []byte("apiVersion: troubleshoot.sh/" + version + "\nkind: HostPreflight\nspec:\n  collectors:\n    - cpu: {}\n"),
			})
			require.NoError(t, err)
			require.NotNil(t, kinds.SupportBundle)
			require.NotNil(t, kinds.HostPreflight)
			require.False(t, kinds.HasUnsupportedTroubleshoot)
			_, err = kotsutil.LoadPreflightFromContents([]byte("apiVersion: troubleshoot.sh/" + version + "\nkind: Preflight\nspec:\n  collectors:\n    - runPod:\n        podSpec:\n          containers:\n            - resources:\n                requests:\n                  memory: not-a-quantity\n"))
			require.ErrorContains(t, err, "quantities must match")
		})
	}
}

func TestRepeatablePreflightValuesAreExpandedBeforeRendering(t *testing.T) {
	for _, version := range []string{"v1beta2", "v1beta3"} {
		t.Run(version, func(t *testing.T) {
			config := `apiVersion: kots.io/v1beta1
kind: Config
metadata: {name: repeat}
spec:
  groups:
    - name: labels
      items:
        - name: label
          type: text
          repeatable: true
          templates:
            - {apiVersion: troubleshoot.sh/VERSION, kind: Preflight, name: repeat, yamlPath: 'spec.analyzers[0]'}
`
			values := "apiVersion: kots.io/v1beta1\nkind: ConfigValues\nmetadata: {name: repeat}\nspec:\n  values:\n    label-1: {value: first, repeatableItem: label}\n    label-2: {value: second, repeatableItem: label}\n"
			preflight := "apiVersion: troubleshoot.sh/" + version + "\nkind: Preflight\nmetadata: {name: repeat}\nspec:\n  analyzers:\n    - clusterVersion:\n        checkName: 'repl{{ ConfigOption \"repl[[ .label ]]\" }}'\n"
			u := &upstreamtypes.Upstream{Type: "replicated", Files: []upstreamtypes.UpstreamFile{
				{Path: "config.yaml", Content: []byte(strings.ReplaceAll(config, "VERSION", version))},
				{Path: "configvalues.yaml", Content: []byte(values)},
				{Path: "preflight.yaml", Content: []byte(preflight)},
			}}
			files, err := RenderKotsKinds(u, &RenderOptions{Namespace: "app", Log: logger.NewCLILogger(nil)})
			require.NoError(t, err)
			kinds, err := kotsutil.KotsKindsFromMap(files)
			require.NoError(t, err)
			require.Len(t, kinds.Preflight.Spec.Analyzers, 2)
			require.ElementsMatch(t, []string{"first", "second"}, []string{kinds.Preflight.Spec.Analyzers[0].ClusterVersion.CheckName, kinds.Preflight.Spec.Analyzers[1].ClusterVersion.CheckName})
		})
	}
}
