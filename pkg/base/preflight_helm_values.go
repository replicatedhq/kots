package base

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kotskinds/pkg/helmchart"
	"helm.sh/helm/v4/pkg/chart/common"
	chartutil "helm.sh/helm/v4/pkg/chart/common/util"
	"helm.sh/helm/v4/pkg/chart/v2/loader"
	"helm.sh/helm/v4/pkg/engine"
	"sigs.k8s.io/yaml"
)

var headerTemplateAction = regexp.MustCompile(`(?s)\{\{.*?\}\}`)

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
		return nil, err
	}
	overrides, err := kotsutil.HelmChartValues(selected)
	if err != nil {
		return nil, err
	}
	if selected.GetNamespace() != "" {
		namespace = selected.GetNamespace()
	}
	// Coalesce defaults and overrides once. Full chart validation belongs to the
	// existing installation path, not rendering this standalone document.
	renderValues, err := chartutil.ToRenderValuesWithSchemaValidation(chart, overrides, common.ReleaseOptions{Name: selected.GetReleaseName(), Namespace: namespace, IsInstall: true, Revision: 1}, nil, true)
	if err != nil {
		return nil, err
	}
	// Keep dependency values, but execute only the standalone spec's template.
	chart.Templates = []*common.File{{Name: "templates/preflight.yaml", Data: content}}
	chart.SetDependencies()
	renderer := engine.Engine{Strict: true}
	rendered, err := renderer.Render(chart, renderValues)
	if err != nil {
		return nil, err
	}
	return []byte(rendered[chart.Name()+"/templates/preflight.yaml"]), nil
}
