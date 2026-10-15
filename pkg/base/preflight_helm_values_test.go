package base

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/stretchr/testify/require"
)

// Both standalone specs and chart Secrets exercise the same templated policy.
const chartValuesPreflight = `apiVersion: troubleshoot.sh/v1beta3
kind: Preflight
metadata:
  annotations:
    kots.io/helm-chart: selected
spec:
  collectors:
    - runPod:
        name: memory-check
        namespace: '{{ .Release.Namespace }}'
        podSpec:
          containers:
            - name: check
              image: busybox:1.36
              resources:
                requests:
                  memory: '{{ .Values.memory }}'
  analyzers:
    - clusterVersion:
        checkName: '{{ .Values.nested.default }}/{{ .Values.nested.override }}'
        strict: true
`

func TestPreflightUsesReferencedChartValues(t *testing.T) {
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	for name, content := range map[string]string{
		"Chart.yaml":                 "apiVersion: v2\nname: values-chart\nversion: 1.0.0\n",
		"values.yaml":                "enabled: false\nmemory: 32Mi\nnested:\n  default: from-default\n",
		"templates/workload.yaml":    `{{ fail "standalone rendering must not execute workload templates" }}`,
		"charts/child/Chart.yaml":    "apiVersion: v2\nname: child\nversion: 1.0.0\n",
		"charts/child/values.yaml":   "proof: from-dependency\n",
		"charts/child/templates/pod": `{{ fail "standalone rendering must not execute dependency templates" }}`,
	} {
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: "values-chart/" + name, Mode: 0600, Size: int64(len(content))}))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	chart := `apiVersion: kots.io/%s
kind: HelmChart
metadata:
  name: selected
spec:
  chart:
    name: values-chart
    chartVersion: 1.0.0
  namespace: chart-namespace
  values:
    memory: 'repl{{ printf "64Mi" }}'
  optionalValues:
    - when: 'true'
      recursiveMerge: true
      values:
        nested:
          override: from-optional
`
	for _, version := range []string{"v1beta1", "v1beta2"} {
		for _, tc := range []struct{ name, doc, err string }{
			{"selected chart", chartValuesPreflight, ""},
			{"dependency values", strings.ReplaceAll(chartValuesPreflight, ".Values.nested.default", ".Values.child.proof"), ""},
			{"missing chart", strings.ReplaceAll(chartValuesPreflight, "helm-chart: selected", "helm-chart: absent"), "no HelmChart"},
			{"missing value", strings.ReplaceAll(chartValuesPreflight, ".Values.memory", ".Values.missing"), "missing"},
			{"disabled conditional", "# outside conditional\n{{ if .Values.enabled }}\n" + chartValuesPreflight + "{{ end }}\n", ""},
		} {
			t.Run(version+"/"+tc.name, func(t *testing.T) {
				u := &upstreamtypes.Upstream{Type: "replicated", Files: []upstreamtypes.UpstreamFile{
					{Path: "preflight.yaml", Content: []byte(tc.doc)},
					{Path: "selected.yaml", Content: []byte(fmt.Sprintf(chart, version))},
					{Path: "other.yaml", Content: []byte(strings.ReplaceAll(strings.ReplaceAll(fmt.Sprintf(chart, version), "name: selected", "name: other"), "64Mi", "256Mi"))},
					{Path: "values-chart-1.0.0.tgz", Content: archive.Bytes()},
				}}
				files, err := RenderKotsKinds(u, &RenderOptions{Namespace: "customer-app", Log: logger.NewCLILogger(nil)})
				if tc.err != "" {
					require.ErrorContains(t, err, tc.err)
					require.ErrorContains(t, err, "preflight.yaml")
					return
				}
				require.NoError(t, err)
				kinds, err := kotsutil.KotsKindsFromMap(files)
				require.NoError(t, err)
				require.False(t, kinds.HasUnsupportedTroubleshoot)
				require.Equal(t, tc.doc, string(u.Files[0].Content), "retain original release template")
				if tc.name == "disabled conditional" {
					require.False(t, kinds.HasPreflights())
					return
				}
				p := kinds.Preflight
				require.Equal(t, "64Mi", p.Spec.Collectors[0].RunPod.PodSpec.Containers[0].Resources.Requests.Memory().String())
				require.Equal(t, "chart-namespace", p.Spec.Collectors[0].RunPod.Namespace)
				expected := "from-default/from-optional"
				if tc.name == "dependency values" {
					expected = "from-dependency/from-optional"
				}
				require.Equal(t, expected, p.Spec.Analyzers[0].ClusterVersion.CheckName)
				require.True(t, p.Spec.Analyzers[0].ClusterVersion.Strict.BoolOrDefaultFalse())
			})
		}
	}
}

func TestBeta3ChartSecretUsesExistingHelmValues(t *testing.T) {
	secret := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: chart-preflight\n  labels:\n    troubleshoot.sh/kind: preflight\nstringData:\n  preflight.yaml: |\n    " + strings.ReplaceAll(strings.TrimSpace(chartValuesPreflight), "\n", "\n    ") + "\n"
	u := &upstreamtypes.Upstream{Name: "preflight-chart", Files: []upstreamtypes.UpstreamFile{
		{Path: "Chart.yaml", Content: []byte("apiVersion: v2\nname: preflight-chart\nversion: 1.0.0\n")},
		{Path: "values.yaml", Content: []byte("memory: 64Mi\nnested: {default: from-default, override: from-optional}\n")},
		{Path: "templates/preflight.yaml", Content: []byte(secret)},
	}}
	b, err := RenderHelm(u, &RenderOptions{Namespace: "customer-app", HelmVersion: "v3"})
	require.NoError(t, err)
	dir := t.TempDir()
	for i, f := range b.Files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, fmt.Sprintf("%d.yaml", i)), f.Content, 0600))
	}
	kinds, err := kotsutil.LoadTSKindsFromPath(dir)
	require.NoError(t, err)
	require.Len(t, kinds.PreflightsV1Beta2, 1)
	p := kinds.PreflightsV1Beta2[0]
	require.Equal(t, "customer-app", p.Spec.Collectors[0].RunPod.Namespace)
	require.Equal(t, "64Mi", p.Spec.Collectors[0].RunPod.PodSpec.Containers[0].Resources.Requests.Memory().String())
	require.Equal(t, "from-default/from-optional", p.Spec.Analyzers[0].ClusterVersion.CheckName)
	require.True(t, p.Spec.Analyzers[0].ClusterVersion.Strict.BoolOrDefaultFalse())
}
