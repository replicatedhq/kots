package base

import (
	"bytes"
	"path"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsv1beta2 "github.com/replicatedhq/kotskinds/apis/kots/v1beta2"
	"github.com/replicatedhq/kotskinds/pkg/helmchart"
	"helm.sh/helm/v4/pkg/chart/common"
	chartutil "helm.sh/helm/v4/pkg/chart/common/util"
	v2chart "helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/chart/v2/loader"
	v2util "helm.sh/helm/v4/pkg/chart/v2/util"
	"helm.sh/helm/v4/pkg/engine"
	"sigs.k8s.io/yaml"
)

var headerTemplateAction = regexp.MustCompile(`(?s)\{\{.*?\}\}`)

// Specs rendered without a chart leave outcome messages to Troubleshoot at runtime.
// Inspect a copy without those messages; never alter the spec we execute.
func hasUnrenderedTroubleshootTemplates(content []byte) bool {
	if !bytes.Contains(content, []byte("{{")) || !bytes.Contains(content, []byte("}}")) {
		return false
	}
	var doc map[string]interface{}
	if yaml.Unmarshal(content, &doc) != nil {
		return true
	}
	spec, _ := doc["spec"].(map[string]interface{})
	for _, section := range []string{"analyzers", "hostAnalyzers"} {
		analyzers, _ := spec[section].([]interface{})
		for _, entry := range analyzers {
			analyzer, _ := entry.(map[string]interface{})
			for _, value := range analyzer {
				config, _ := value.(map[string]interface{})
				outcomes, _ := config["outcomes"].([]interface{})
				for _, entry := range outcomes {
					outcome, _ := entry.(map[string]interface{})
					for _, value := range outcome {
						fields, _ := value.(map[string]interface{})
						delete(fields, "message")
					}
				}
			}
		}
	}
	remaining, err := yaml.Marshal(doc)
	return err != nil || (bytes.Contains(remaining, []byte("{{")) && bytes.Contains(remaining, []byte("}}")))
}

// Keep template-only documents (including surrounding YAML comments) in scope.
func isHelmScopeFragment(content []byte) bool {
	stripped := headerTemplateAction.ReplaceAll(content, nil)
	var doc interface{}
	return !bytes.Equal(content, stripped) && yaml.Unmarshal(stripped, &doc) == nil && doc == nil
}

// Read the header without evaluating Helm conditionals around the spec.
func isBeta3Preflight(content []byte) bool {
	if !kotsutil.IsV1Beta3TroubleshootDoc(content) {
		return false
	}
	header := strings.SplitN(strings.ReplaceAll(string(content), "\r\n", "\n"), "\nspec:", 2)[0]
	var meta OverlySimpleGVK
	return yaml.Unmarshal(headerTemplateAction.ReplaceAll([]byte(header), nil), &meta) == nil && meta.Kind == "Preflight"
}

// Render once per installed chart, as EC does. The spec can select charts with
// .Chart.Name conditions; chart values are never combined across charts.
func renderTroubleshootWithCharts(content []byte, kinds *kotsutil.KotsKinds, files []upstreamtypes.UpstreamFile, namespace string) ([]byte, error) {
	var beta1 []kotsv1beta1.HelmChart
	var beta2 []kotsv1beta2.HelmChart
	if kinds.V1Beta1HelmCharts != nil {
		beta1 = kinds.V1Beta1HelmCharts.Items
	}
	if kinds.V1Beta2HelmCharts != nil {
		beta2 = kinds.V1Beta2HelmCharts.Items
	}
	var charts []helmchart.HelmChartInterface
	for _, h := range kotsutil.FilterV1Beta1ChartsWithV1Beta2Charts(beta1, beta2) {
		if !h.Spec.Exclude.BoolOrDefaultFalse() {
			charts = append(charts, &h)
		}
	}
	for _, h := range beta2 {
		if !h.Spec.Exclude.BoolOrDefaultFalse() {
			charts = append(charts, &h)
		}
	}
	if len(charts) == 0 {
		if hasUnrenderedTroubleshootTemplates(content) {
			return nil, errors.New("preflight uses Helm templates but no enabled Helm charts are available")
		}
		return content, nil
	}
	var rendered [][]byte
	for _, chart := range charts {
		doc, err := renderTroubleshootWithChart(content, chart, files, namespace)
		if err != nil {
			return nil, errors.Wrapf(err, "render preflight for chart %s", chart.GetChartName())
		}
		rendered = append(rendered, doc)
	}
	return bytes.Join(rendered, []byte("\n---\n")), nil
}

func renderTroubleshootWithChart(content []byte, selected helmchart.HelmChartInterface, files []upstreamtypes.UpstreamFile, namespace string) ([]byte, error) {
	archive, err := FindHelmChartArchiveInRelease(files, selected)
	if err != nil {
		return nil, err
	}
	chart, err := loader.LoadArchive(bytes.NewReader(archive))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load Helm chart archive")
	}
	overrides, err := kotsutil.HelmChartValues(selected)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get Helm chart values")
	}
	if selected.GetNamespace() != "" {
		namespace = selected.GetNamespace()
	}
	if err := v2util.ProcessDependencies(chart, overrides); err != nil {
		return nil, errors.Wrap(err, "failed to process Helm chart dependencies")
	}
	// Coalesce defaults and overrides once. Full chart validation belongs to the
	// existing installation path, not rendering this standalone document.
	renderValues, err := chartutil.ToRenderValuesWithSchemaValidation(chart, overrides, common.ReleaseOptions{Name: selected.GetDirName(), Namespace: namespace, IsInstall: true, Revision: 1}, nil, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build preflight render values")
	}
	// Helm parses underscore-prefixed files for definitions without executing
	// them. Retain helpers from the entire chart tree, including library charts.
	var keepDefinitions func(*v2chart.Chart)
	keepDefinitions = func(c *v2chart.Chart) {
		for _, file := range c.Templates {
			file.Name = path.Join(path.Dir(file.Name), "_"+path.Base(file.Name))
		}
		for _, dependency := range c.Dependencies() {
			keepDefinitions(dependency)
		}
	}
	keepDefinitions(chart)
	chart.Templates = append(chart.Templates, &common.File{Name: "templates/preflight.yaml", Data: content})
	rendered, err := engine.Render(chart, renderValues)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render preflight template")
	}
	return []byte(rendered[chart.Name()+"/templates/preflight.yaml"]), nil
}
