package base

import (
	"bytes"
	"path"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
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

// Unannotated specs leave analyzer outcome messages to Troubleshoot at runtime.
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

// Only parse the root header: the spec may not be valid YAML until Helm renders
// it. Ignore conditional wrappers when reading the literal chart reference.
func troubleshootHelmChart(content []byte) string {
	if !kotsutil.IsV1Beta3TroubleshootDoc(content) {
		return ""
	}
	header := strings.SplitN(strings.ReplaceAll(string(content), "\r\n", "\n"), "\nspec:", 2)[0]
	var meta OverlySimpleGVK
	if yaml.Unmarshal(headerTemplateAction.ReplaceAll([]byte(header), nil), &meta) != nil || meta.Kind != "Preflight" {
		return ""
	}
	ref, _ := meta.Metadata.Annotations["kots.io/helm-chart"].(string)
	return ref
}

func renderTroubleshootWithChart(content []byte, ref string, kinds *kotsutil.KotsKinds, files []upstreamtypes.UpstreamFile, namespace string) ([]byte, error) {
	var selected helmchart.HelmChartInterface
	if kinds.V1Beta1HelmCharts != nil {
		for i := range kinds.V1Beta1HelmCharts.Items {
			if h := &kinds.V1Beta1HelmCharts.Items[i]; h.Name == ref {
				selected = h
			}
		}
	}
	if kinds.V1Beta2HelmCharts != nil {
		for i := range kinds.V1Beta2HelmCharts.Items {
			if h := &kinds.V1Beta2HelmCharts.Items[i]; h.Name == ref {
				selected = h // As elsewhere, beta2 supersedes beta1 with the same name.
			}
		}
	}
	if selected == nil {
		return nil, errors.New("no HelmChart with that metadata.name exists in this release")
	}
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
