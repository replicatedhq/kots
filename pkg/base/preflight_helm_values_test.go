package base

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsadmconfig"
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
        name: memory-check-{{ .Release.Name }}
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
    - textAnalyze:
        checkName: Runtime message
        strict: true
        regexGroups: '(?P<version>old)'
        outcomes:
          - fail:
              when: 'version = old'
              message: '{{ "{{" }} if eq .version "old" {{ "}}" }}Unsupported version {{ "{{" }} printf "%s" (.version | upper) {{ "}}" }}{{ "{{" }} end {{ "}}" }}'
`

func requireRuntimeMessagePreserved(t *testing.T, kinds *kotsutil.KotsKinds) {
	t.Helper()
	require.False(t, kinds.HasUnsupportedTroubleshoot)
	analyzer := kinds.Preflight.Spec.Analyzers[1].TextAnalyze
	require.True(t, analyzer.Strict.BoolOrDefaultFalse())
	require.Equal(t, `{{ if eq .version "old" }}Unsupported version {{ printf "%s" (.version | upper) }}{{ end }}`, analyzer.Outcomes[0].Fail.Message)
}

func TestPreflightUsesReferencedChartValues(t *testing.T) {
	archive := releaseChart(t, "values-chart", map[string]string{
		"Chart.yaml":                            "apiVersion: v2\nname: values-chart\nversion: 1.0.0\ndependencies:\n  - name: child\n    version: 1.0.0\n    repository: file://charts/child\n    import-values: [settings]\n",
		"values.yaml":                           "enabled: false\nmemory: 32Mi\nnested:\n  default: from-default\n",
		"templates/workload.yaml":               `{{ fail "standalone rendering must not execute workload templates" }}`,
		"templates/_helpers.tpl":                `{{ define "values.memory" }}{{ .Values.memory }}{{ end }}`,
		"charts/child/Chart.yaml":               "apiVersion: v2\nname: child\nversion: 1.0.0\n",
		"charts/child/values.yaml":              "proof: from-dependency\nexports:\n  settings:\n    imported: from-import\n",
		"charts/child/templates/pod":            `{{ fail "standalone rendering must not execute dependency templates" }}`,
		"charts/child/templates/_helpers.tpl":   `{{ define "child.memory" }}{{ .Values.memory }}{{ end }}`,
		"charts/library/Chart.yaml":             "apiVersion: v2\nname: library\nversion: 1.0.0\ntype: library\n",
		"charts/library/templates/_helpers.tpl": `{{ define "library.memory" }}{{ .Values.memory }}{{ end }}`,
	})
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
			{"root helper", strings.ReplaceAll(chartValuesPreflight, ".Values.memory", `include "values.memory" .`), ""},
			{"dependency helper", strings.ReplaceAll(chartValuesPreflight, ".Values.memory", `include "child.memory" .`), ""},
			{"library helper", strings.ReplaceAll(chartValuesPreflight, ".Values.memory", `template "library.memory" .`), ""},
			{"separate closing scope", "{{ if .Values.memory }}\n" + chartValuesPreflight + "\n---\n{{ end }}\n", ""},
			{"separate scopes", "{{ if .Values.memory }}\n---\n" + chartValuesPreflight + "\n---\n{{ end }}\n", ""},
			{"disabled separate scopes", "{{ if .Values.enabled }}\n---\n" + chartValuesPreflight + "\n---\n{{ end }}\n", ""},
			{"dependency values", strings.ReplaceAll(chartValuesPreflight, ".Values.nested.default", ".Values.child.proof"), ""},
			{"imported values", strings.ReplaceAll(chartValuesPreflight, ".Values.nested.default", ".Values.imported"), ""},
			{"multiple documents", "{{ if .Values.memory }}\n" + chartValuesPreflight + "\n---\n" + chartValuesPreflight + "{{ end }}\n", ""},
			{"chart in same file", fmt.Sprintf(chart, version) + "\n---\n" + chartValuesPreflight, ""},
			{"configmap in same file", "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: retained-config\ndata: {proof: retained}\n---\n" + chartValuesPreflight, ""},
			{"beta2 in same file", chartValuesPreflight + "\n---\napiVersion: troubleshoot.sh/v1beta2\nkind: Preflight\nspec:\n  analyzers:\n    - textAnalyze:\n        checkName: Beta2 runtime\n        strict: true\n        outcomes:\n          - fail: {message: 'Unsupported version {{ .version }}'}\n", ""},
			{"cross-document variable", "{{ $memory := .Values.memory }}\n" + chartValuesPreflight + "\n---\n" + strings.ReplaceAll(chartValuesPreflight, ".Values.memory", "$memory"), ""},
			{"missing chart", strings.ReplaceAll(chartValuesPreflight, "helm-chart: selected", "helm-chart: absent"), "no HelmChart"},
			{"missing value", strings.ReplaceAll(chartValuesPreflight, ".Values.memory", `.Values.missing | required "missing memory"`), "missing memory"},
			{"broken template", chartValuesPreflight + "{{ if }}", "missing value for if"},
			{"invalid rendered YAML", strings.ReplaceAll(chartValuesPreflight, "'{{ .Values.memory }}'", `{{ printf "[" }}`), "invalid rendered YAML"},
			{"configuration pending", strings.ReplaceAll(chartValuesPreflight, ".Values.memory", `.Values.memory | required "memory is required"`), ""},
			{"configuration supplied", strings.ReplaceAll(chartValuesPreflight, ".Values.memory", `.Values.memory | required "memory is required"`), ""},
			{"default for missing value", strings.ReplaceAll(chartValuesPreflight, ".Values.memory", `.Values.optional | default "64Mi"`), ""},
			{"disabled conditional", "# outside conditional\n{{ if .Values.enabled }}\n" + chartValuesPreflight + "{{ end }}\n", ""},
			{"disabled missing conditional", "{{ if .Values.optional }}\n" + chartValuesPreflight + "{{ end }}\n", ""},
		} {
			t.Run(version+"/"+tc.name, func(t *testing.T) {
				u := &upstreamtypes.Upstream{Type: "replicated", Files: []upstreamtypes.UpstreamFile{
					{Path: "preflight.yaml", Content: []byte(tc.doc)},
					{Path: "selected.yaml", Content: []byte(fmt.Sprintf(chart, version))},
					{Path: "other.yaml", Content: []byte(strings.ReplaceAll(strings.ReplaceAll(fmt.Sprintf(chart, version), "name: selected", "name: other"), "64Mi", "256Mi"))},
					{Path: "values-chart-1.0.0.tgz", Content: archive.Content},
				}}
				if tc.name == "chart in same file" {
					u.Files[1].Content = nil
				}
				if tc.err != "" {
					u.Files = append(u.Files, upstreamtypes.UpstreamFile{Path: "supported.yaml", Content: []byte("apiVersion: troubleshoot.sh/v1beta2\nkind: Preflight\nspec:\n  analyzers:\n    - clusterVersion:\n        checkName: retained\n        strict: true\n")})
				}
				if strings.HasPrefix(tc.name, "configuration") {
					u.Files[1].Content = []byte(strings.ReplaceAll(string(u.Files[1].Content), `printf "64Mi"`, `ConfigOption "memory"`))
					u.Files = append(u.Files, upstreamtypes.UpstreamFile{Path: "config.yaml", Content: []byte("apiVersion: kots.io/v1beta1\nkind: Config\nmetadata: {name: required}\nspec:\n  groups:\n    - name: settings\n      items:\n        - {name: memory, type: text, required: true}\n")})
					if tc.name == "configuration supplied" {
						u.Files = append(u.Files, upstreamtypes.UpstreamFile{Path: "configvalues.yaml", Content: []byte("apiVersion: kots.io/v1beta1\nkind: ConfigValues\nmetadata: {name: required}\nspec:\n  values:\n    memory: {value: 64Mi}\n")})
					}
				}
				var logs bytes.Buffer
				opts := &RenderOptions{Namespace: "customer-app", Log: logger.NewCLILogger(&logs)}
				files, err := RenderKotsKinds(u, opts)
				require.NoError(t, err)
				if tc.err != "" {
					require.Contains(t, logs.String(), tc.err)
					require.Contains(t, logs.String(), "preflight.yaml")
					dir := t.TempDir()
					require.NoError(t, kotsutil.WriteKotsKinds(files, dir))
					kinds, err := kotsutil.LoadKotsKinds(dir)
					require.NoError(t, err)
					require.True(t, kinds.HasUnsupportedTroubleshoot)
					require.Len(t, kinds.Preflight.Spec.Analyzers, 1)
					require.Equal(t, "retained", kinds.Preflight.Spec.Analyzers[0].ClusterVersion.CheckName)
					require.True(t, kinds.Preflight.Spec.Analyzers[0].ClusterVersion.Strict.BoolOrDefaultFalse())
					return
				}
				require.NoError(t, err)
				kinds, err := kotsutil.KotsKindsFromMap(files)
				require.NoError(t, err)
				require.False(t, kinds.HasUnsupportedTroubleshoot)
				require.Equal(t, tc.doc, string(u.Files[0].Content), "retain original release template")
				if tc.name == "configuration pending" {
					needsConfig, err := kotsadmconfig.NeedsConfiguration("", 0, false, kinds, opts.RegistrySettings)
					require.NoError(t, err)
					require.True(t, needsConfig, "pull must reach the configuration screen")
					require.False(t, kinds.HasPreflights(), "checks must wait for required configuration")
					return
				}
				if strings.HasPrefix(tc.name, "disabled") {
					require.False(t, kinds.HasPreflights())
					return
				}
				dir := t.TempDir()
				require.NoError(t, kotsutil.WriteKotsKinds(files, dir))
				reloaded, err := kotsutil.LoadKotsKinds(dir)
				require.NoError(t, err)
				for _, kinds := range []*kotsutil.KotsKinds{kinds, reloaded} {
					p := kinds.Preflight
					if tc.name == "beta2 in same file" {
						require.Equal(t, "Unsupported version {{ .version }}", p.Spec.Analyzers[0].TextAnalyze.Outcomes[0].Fail.Message)
						require.True(t, p.Spec.Analyzers[0].TextAnalyze.Strict.BoolOrDefaultFalse())
						p.Spec.Analyzers = p.Spec.Analyzers[1:]
					}
					releaseName := "values-chart"
					if version == "v1beta1" {
						releaseName = "selected"
					}
					require.Equal(t, "memory-check-"+releaseName, p.Spec.Collectors[0].RunPod.Name)
					require.Equal(t, "64Mi", p.Spec.Collectors[0].RunPod.PodSpec.Containers[0].Resources.Requests.Memory().String())
					require.Equal(t, "chart-namespace", p.Spec.Collectors[0].RunPod.Namespace)
					expected := "from-default/from-optional"
					if tc.name == "dependency values" {
						expected = "from-dependency/from-optional"
					}
					if tc.name == "imported values" {
						expected = "from-import/from-optional"
					}
					if tc.name == "multiple documents" || tc.name == "cross-document variable" {
						require.Len(t, p.Spec.Analyzers, 4)
						require.Equal(t, "64Mi", p.Spec.Collectors[1].RunPod.PodSpec.Containers[0].Resources.Requests.Memory().String())
					}
					require.Equal(t, expected, p.Spec.Analyzers[0].ClusterVersion.CheckName)
					require.True(t, p.Spec.Analyzers[0].ClusterVersion.Strict.BoolOrDefaultFalse())
					requireRuntimeMessagePreserved(t, kinds)
				}
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
	require.Equal(t, "memory-check-preflight-chart", p.Spec.Collectors[0].RunPod.Name)
	require.Equal(t, "customer-app", p.Spec.Collectors[0].RunPod.Namespace)
	require.Equal(t, "64Mi", p.Spec.Collectors[0].RunPod.PodSpec.Containers[0].Resources.Requests.Memory().String())
	require.Equal(t, "from-default/from-optional", p.Spec.Analyzers[0].ClusterVersion.CheckName)
	require.True(t, p.Spec.Analyzers[0].ClusterVersion.Strict.BoolOrDefaultFalse())
	requireRuntimeMessagePreserved(t, &kotsutil.KotsKinds{Preflight: &p})
}
