package base

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	kotsconfig "github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/kotsadmconfig"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsv1beta2 "github.com/replicatedhq/kotskinds/apis/kots/v1beta2"
	kotsscheme "github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kotskinds/pkg/helmchart"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	v2chart "helm.sh/helm/v4/pkg/chart/v2"
	"k8s.io/client-go/kubernetes/scheme"
	applicationv1beta1 "sigs.k8s.io/application/api/v1beta1"
	"sigs.k8s.io/yaml"
)

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
	troubleshootscheme.AddToScheme(scheme.Scheme)
	velerov1.AddToScheme(scheme.Scheme)
	applicationv1beta1.AddToScheme(scheme.Scheme)
}

type Document struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
}

func renderReplicated(u *upstreamtypes.Upstream, renderOptions *RenderOptions, renderedKotsKinds *kotsutil.KotsKinds) (*Base, []Base, error) {
	commonBase := Base{
		Files: []BaseFile{},
		Bases: []Base{},
	}

	builder, _, err := NewConfigContextTemplateBuilder(u, renderOptions)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create new config context template builder")
	}

	for _, upstreamFile := range u.Files {
		if renderOptions.ExcludeKotsKinds {
			// kots kinds are not expected to be valid yaml after builder.RenderTemplate
			// this will prevent errors later from ShouldBeIncludedInBaseKustomization
			newContent := [][]byte{}
			isKotsKind := false
			for _, doc := range util.ConvertToSingleDocs(upstreamFile.Content) {
				file := BaseFile{Path: upstreamFile.Path, Content: doc}
				// ignore the error here, we will catch it later in ShouldBeIncludedInBaseKustomization
				if ok, _ := file.IsKotsKind(); ok {
					isKotsKind = true
				} else {
					newContent = append(newContent, doc)
				}
			}
			if isKotsKind && len(newContent) == 0 {
				continue
			}
			upstreamFile.Content = bytes.Join(newContent, []byte("\n---\n"))
		}

		c, err := processVariadicConfig(&upstreamFile, renderedKotsKinds.Config, renderOptions.Log)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to process variadic config in file %s", upstreamFile.Path)
		}
		upstreamFile.Content = c

		baseFile, err := upstreamFileToBaseFile(upstreamFile, *builder, renderOptions.Log)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert upstream file %s to base", upstreamFile.Path)
		}

		baseFiles := convertToSingleDocBaseFiles([]BaseFile{baseFile})
		for _, f := range baseFiles {
			include, err := f.ShouldBeIncludedInBaseKustomization(renderOptions.ExcludeKotsKinds)
			if err != nil {
				if _, ok := err.(ParseError); !ok {
					return nil, nil, errors.Wrapf(err, "failed to determine if file %s should be included in base", f.Path)
				}
			}
			if include {
				commonBase.Files = append(commonBase.Files, f)
			} else if err != nil {
				f.Error = err
				commonBase.ErrorFiles = append(commonBase.ErrorFiles, f)
			}
		}
	}

	// render helm charts that were specified
	// we just inject them into u.Files
	// NOTE: we only render v1beta1 HelmCharts to base
	kotsV1Beta1HelmCharts, err := findAllKotsV1Beta1HelmCharts(u.Files, *builder, renderOptions.Log)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to find helm charts")
	}

	helmBases := []Base{}
	for _, kotsHelmChart := range kotsV1Beta1HelmCharts {
		helmBase, err := renderReplicatedHelmChart(&kotsHelmChart, u.Files, renderOptions, builder)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to render helm chart %s", kotsHelmChart.Name)
		} else if helmBase == nil {
			continue
		}

		renderedHelmBase, err := renderReplicatedHelmBase(u, renderOptions, *helmBase, *builder)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to render helm chart base %s", helmBase.Path)
		}

		if kotsHelmChart.Spec.UseHelmInstall {
			helmBases = append(helmBases, extractHelmBases(*renderedHelmBase)...)
		} else {
			commonBase.Bases = append(commonBase.Bases, *renderedHelmBase)
		}
	}

	return &commonBase, helmBases, nil
}

func renderKotsKinds(u *upstreamtypes.Upstream, renderOptions *RenderOptions) (map[string][]byte, error) {
	renderedKotsKindsMap := make(map[string][]byte)
	type chartPreflight struct {
		BaseFile
		ref string
	}
	var chartPreflights []chartPreflight

	builder, itemValues, err := NewConfigContextTemplateBuilder(u, renderOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new config context template builder")
	}

	tkk, err := getTemplatingKotsKinds(u)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find config file")
	}

	versionInfo := template.VersionInfoFromInstallationSpec(renderOptions.Sequence, renderOptions.IsAirgap, tkk.Installation.Spec)
	appInfo := template.ApplicationInfo{Slug: renderOptions.AppSlug}

	renderedConfig, err := kotsconfig.TemplateConfigObjects(tkk.Config, itemValues, tkk.License, &tkk.KotsApplication, renderOptions.RegistrySettings, &versionInfo, &appInfo, tkk.IdentityConfig, util.PodNamespace, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to template config objects")
	}
	renderSpec := func(file BaseFile, expandConfig bool) ([]byte, error) {
		if expandConfig {
			var err error
			file.Content, err = processVariadicConfig(&upstreamtypes.UpstreamFile{Content: file.Content, Path: file.Path}, renderedConfig, renderOptions.Log)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to process variadic config in %s", file.Path)
			}
		}
		rendered, err := builder.RenderTemplate(file.Path, string(file.Content))
		return []byte(rendered), errors.Wrapf(err, "failed to render file %s", file.Path)
	}

	for _, upstreamFile := range u.Files {
		docs := util.ConvertToSingleDocs(upstreamFile.Content)
		// HelmChart rendering historically retained ordinary sibling documents.
		// Render each document once while preserving that mixed-file output.
		hasHelmChart := false
		chartRef := ""
		for _, doc := range docs {
			if ref := troubleshootHelmChart(doc); ref != "" {
				if chartRef != "" && chartRef != ref {
					return nil, errors.Errorf("%s: preflights in one template file must reference the same HelmChart", upstreamFile.Path)
				}
				chartRef = ref
			}
			var gvk OverlySimpleGVK
			if err := yaml.Unmarshal(doc, &gvk); err == nil && gvk.Kind == "HelmChart" && (gvk.APIVersion == "kots.io/v1beta1" || gvk.APIVersion == "kots.io/v1beta2") {
				hasHelmChart = true
			}
		}
		var chartDocs [][]byte
		for _, doc := range docs {
			if troubleshootHelmChart(doc) != "" || (chartRef != "" && isHelmScopeFragment(doc)) {
				chartDocs = append(chartDocs, doc)
				continue
			}
			isBeta3 := kotsutil.IsV1Beta3TroubleshootDoc(doc)
			if isBeta3 {
				// Render KOTS expressions first. Like the Troubleshoot CLI, raw
				// beta3 templates need an explicit values source. Already-rendered
				// specs are loaded directly and never inspected for expressions.
				rendered, err := renderSpec(BaseFile{Path: upstreamFile.Path, Content: doc}, true)
				if err != nil {
					return nil, err
				}
				doc = rendered
				if bytes.Contains(doc, []byte("{{")) && bytes.Contains(doc, []byte("}}")) {
					doc = kotsutil.UnsupportedTroubleshootWarning()
				}
			}
			gvk := OverlySimpleGVK{}
			if err := yaml.Unmarshal(doc, &gvk); err != nil {
				if !hasHelmChart {
					continue
				}
				// Whole-file HelmChart rendering retained these bytes too. A
				// sibling may only become valid YAML after textual rendering.
				gvk = OverlySimpleGVK{}
			}

			isKotsKind := kotsutil.IsKotsKind(gvk.APIVersion, gvk.Kind)
			if !isKotsKind && !hasHelmChart {
				continue
			}

			gvkString := fmt.Sprintf("%s, Kind=%s", gvk.APIVersion, gvk.Kind)

			switch gvkString {
			case "kots.io/v1beta1, Kind=Installation", "kots.io/v1beta1, Kind=License",
				"kots.io/v1beta1, Kind=IdentityConfig", "kots.io/v1beta1, Kind=ConfigValues":
				// These manifests do not need rendering.

			case "kots.io/v1beta1, Kind=Config":
				// Use the rendered Config instead of the upstream.
				kotsKinds := kotsutil.KotsKinds{Config: renderedConfig}
				config, err := kotsKinds.Marshal("kots.io", "v1beta1", "Config")
				if err != nil {
					return nil, errors.Wrap(err, "failed to marshal rendered config")
				}
				doc = []byte(config)

			case "kots.io/v1beta1, Kind=HelmChart", "kots.io/v1beta2, Kind=HelmChart":
				chartBuilder := helmChartTemplateBuilder(doc, u.Files, *builder)
				helmchart, err := chartBuilder.RenderTemplate(upstreamFile.Path, string(doc))
				if err != nil {
					return nil, errors.Wrapf(err, "failed to render file %s", upstreamFile.Path)
				}
				doc = []byte(helmchart)

			default:
				if !isBeta3 {
					doc, err = renderSpec(BaseFile{Path: upstreamFile.Path, Content: doc}, isKotsKind)
					if err != nil {
						return nil, err
					}
				}
			}

			if existing, exists := renderedKotsKindsMap[upstreamFile.Path]; exists {
				doc = bytes.Join([][]byte{existing, doc}, []byte("\n---\n"))
			}

			renderedKotsKindsMap[upstreamFile.Path] = doc
		}
		if len(chartDocs) > 0 {
			// Render annotated fragments together so Helm scopes can span documents.
			chartPreflights = append(chartPreflights, chartPreflight{BaseFile{Path: upstreamFile.Path, Content: bytes.Join(chartDocs, []byte("\n---\n"))}, chartRef})
		}
	}

	if len(chartPreflights) > 0 {
		charts, err := kotsutil.KotsKindsFromMap(renderedKotsKindsMap)
		if err != nil {
			return nil, err
		}
		needsConfig, err := kotsadmconfig.NeedsConfiguration(renderOptions.AppSlug, renderOptions.Sequence, renderOptions.IsAirgap, charts, renderOptions.RegistrySettings)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check configuration before rendering preflights")
		}
		if needsConfig {
			return renderedKotsKindsMap, nil
		}
		for _, file := range chartPreflights {
			ref := file.ref
			rendered, err := renderSpec(file.BaseFile, true)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to render KOTS templates in %s", file.Path)
			}
			doc, err := renderTroubleshootWithChart(rendered, ref, charts, u.Files, renderOptions.Namespace)
			if err != nil {
				renderOptions.Log.Error(errors.Wrapf(err, "skipping %s: cannot render with values from HelmChart %q", file.Path, ref))
				doc = kotsutil.UnsupportedTroubleshootWarning()
			} else {
				var nonempty [][]byte
				for _, part := range util.ConvertToSingleDocs(doc) {
					var parsed OverlySimpleGVK
					if err := yaml.Unmarshal(part, &parsed); err != nil {
						return nil, errors.Wrapf(err, "invalid rendered YAML in %s", file.Path)
					}
					if troubleshootHelmChart(part) != "" {
						nonempty = append(nonempty, part)
					}
				}
				doc = bytes.Join(nonempty, []byte("\n---\n"))
			}
			if len(doc) == 0 {
				continue
			}
			if existing := renderedKotsKindsMap[file.Path]; len(existing) > 0 {
				doc = bytes.Join([][]byte{existing, doc}, []byte("\n---\n"))
			}
			renderedKotsKindsMap[file.Path] = doc
		}
	}
	return renderedKotsKindsMap, nil
}

func extractHelmBases(b Base) []Base {
	bases := []Base{}
	for _, sub := range b.Bases {
		sub.Path = path.Join(b.Path, sub.Path)
		bases = append(bases, extractHelmBases(sub)...)
	}
	b.Bases = nil
	return append([]Base{b}, bases...)
}

func renderReplicatedHelmChart(kotsHelmChart *kotsv1beta1.HelmChart, upstreamFiles []upstreamtypes.UpstreamFile, renderOptions *RenderOptions, builder *template.Builder) (*Base, error) {
	if !kotsHelmChart.Spec.Exclude.IsEmpty() {
		boolVal, err := kotsHelmChart.Spec.Exclude.Boolean()
		if err != nil {
			renderOptions.Log.Error(errors.Errorf("Kots.io/v1beta1 HelmChart %s rendered exclude is not parseable as bool, value = %s. Not excluding chart.", kotsHelmChart.Name, kotsHelmChart.Spec.Exclude.String()))

			return nil, errors.Wrap(err, "failed to parse helm chart exclude")
		}

		if boolVal {
			return nil, nil
		}
	}

	// Include this chart
	archive, err := FindHelmChartArchiveInRelease(upstreamFiles, kotsHelmChart)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find helm chart archive in release")
	}

	tmpFile, err := ioutil.TempFile("", "kots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp file")
	}
	defer os.RemoveAll(tmpFile.Name())

	_, err = io.Copy(tmpFile, bytes.NewReader(archive))
	if err != nil {
		return nil, errors.Wrap(err, "failed to copy chart to temp file")
	}

	helmUpstream, err := chartArchiveToSparseUpstream(tmpFile.Name())
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch helm dependency")
	}
	helmUpstream.Name = kotsHelmChart.GetDirName()

	helmValues, err := kotsutil.HelmChartValues(kotsHelmChart)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render local values for chart")
	}

	helmBase, err := RenderHelm(helmUpstream, &RenderOptions{
		SplitMultiDocYAML: true,
		Namespace:         kotsHelmChart.Spec.Namespace,
		HelmVersion:       kotsHelmChart.Spec.HelmVersion,
		HelmValues:        helmValues,
		ExcludeKotsKinds:  renderOptions.ExcludeKotsKinds,
		Log:               nil,
		UseHelmInstall:    kotsHelmChart.Spec.UseHelmInstall,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to render helm chart in upstream")
	}

	helmBase.Path = path.Join("charts", helmUpstream.Name)
	return helmBase, nil
}

// HelmValue chart context belongs to the kots.io HelmChart manifest, matching EC.
// This pass processes native Helm output, outside that context, so HelmValue
// retains its empty fallback; chart templates should use native Helm .Values.
func renderReplicatedHelmBase(u *upstreamtypes.Upstream, renderOptions *RenderOptions, helmBase Base, builder template.Builder) (*Base, error) {
	helmBaseFiles, helmBaseYAMLErrorFiles := []BaseFile{}, []BaseFile{}
	for _, helmBaseFile := range helmBase.Files {
		upstreamFile := upstreamtypes.UpstreamFile{
			Path:    helmBaseFile.Path,
			Content: helmBaseFile.Content,
		}
		u.Files = append(u.Files, upstreamFile)

		baseFile, err := upstreamFileToBaseFile(upstreamFile, builder, renderOptions.Log)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert upstream file %s to base", upstreamFile.Path)
		}

		// this is a little bit of an abuse of the next function
		include, err := helmBaseFile.ShouldBeIncludedInBaseKustomization(false)
		if err != nil {
			if _, ok := err.(ParseError); !ok {
				return nil, errors.Wrapf(err, "failed to determine if file %s should be included in base", upstreamFile.Path)
			}
		}

		if include {
			helmBaseFiles = append(helmBaseFiles, baseFile)
		} else if err != nil {
			baseFile.Error = err
			helmBaseYAMLErrorFiles = append(helmBaseYAMLErrorFiles, baseFile)
		}
	}

	helmBaseBases := []Base{}
	for _, helmBase := range helmBase.Bases {
		renderedHelmBase, err := renderReplicatedHelmBase(u, renderOptions, helmBase, builder)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to render helm chart %s base", helmBase.Path)
		}

		helmBaseBases = append(helmBaseBases, *renderedHelmBase)
	}

	return &Base{
		Path:            helmBase.Path,
		Namespace:       helmBase.Namespace,
		Files:           helmBaseFiles,
		AdditionalFiles: helmBase.AdditionalFiles,
		Bases:           helmBaseBases,
		ErrorFiles:      helmBaseYAMLErrorFiles,
	}, nil
}

func upstreamFileToBaseFile(upstreamFile upstreamtypes.UpstreamFile, builder template.Builder, log *logger.CLILogger) (BaseFile, error) {
	rendered, err := builder.RenderTemplate(upstreamFile.Path, string(upstreamFile.Content))
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to render file %s. Contents are %s", upstreamFile.Path, upstreamFile.Content))
		return BaseFile{}, errors.Wrap(err, "failed to render file template")
	}

	return BaseFile{
		Path:    upstreamFile.Path,
		Content: []byte(rendered),
	}, nil
}

// findAllKotsV1Beta1HelmCharts finds all kotsv1beta1.HelmChart objects in the upstream
// and will filter out any charts where a cooresponding kotsv1beta2.HelmChart exists
// because the kotsv1beta2.HelmChart will be used instead
func findAllKotsV1Beta1HelmCharts(upstreamFiles []upstreamtypes.UpstreamFile, builder template.Builder, log *logger.CLILogger) ([]kotsv1beta1.HelmChart, error) {
	kotsV1Beta1HelmCharts := []kotsv1beta1.HelmChart{}
	kotsV1Beta2HelmCharts := []kotsv1beta2.HelmChart{}

	// Preserve the existing whole-file eligibility check for chart discovery.
	for _, upstreamFile := range upstreamFiles {
		var helmChartVersion string
		if kotsutil.IsApiVersionKind(upstreamFile.Content, "kots.io/v1beta1", "HelmChart") {
			helmChartVersion = "v1beta1"
		} else if kotsutil.IsApiVersionKind(upstreamFile.Content, "kots.io/v1beta2", "HelmChart") {
			helmChartVersion = "v1beta2"
		} else {
			continue
		}

		chartBuilder := helmChartTemplateBuilder(upstreamFile.Content, upstreamFiles, builder)
		baseFile, err := upstreamFileToBaseFile(upstreamFile, chartBuilder, log)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert upstream file %s to base", upstreamFile.Path)
		}

		switch helmChartVersion {
		case "v1beta1":
			helmChart, err := kotsutil.LoadV1Beta1HelmChartFromContents(baseFile.Content)
			if err != nil {
				fmt.Printf("Failed v1beta1 HelmChart contents:\n%s\n", string(baseFile.Content))
				return nil, errors.Wrapf(err, "failed to parse rendered v1beta1 HelmChart %s", baseFile.Path)
			}
			kotsV1Beta1HelmCharts = append(kotsV1Beta1HelmCharts, *helmChart)
		case "v1beta2":
			helmChart, err := kotsutil.LoadV1Beta2HelmChartFromContents(baseFile.Content)
			if err != nil {
				fmt.Printf("Failed v1beta2 HelmChart contents:\n%s\n", string(baseFile.Content))
				return nil, errors.Wrapf(err, "failed to parse rendered v1beta2 HelmChart %s", baseFile.Path)
			}
			kotsV1Beta2HelmCharts = append(kotsV1Beta2HelmCharts, *helmChart)
		}
	}

	// filter out any charts where a cooresponding v1beta2 HelmChart exists
	filteredCharts := kotsutil.FilterV1Beta1ChartsWithV1Beta2Charts(kotsV1Beta1HelmCharts, kotsV1Beta2HelmCharts)

	return filteredCharts, nil
}

func tryGetConfigFromFileContent(content []byte, log *logger.CLILogger) *kotsv1beta1.Config {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		log.Debug("Failed to parse file while looking for config: %v", err)
		return nil
	}

	if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Config" {
		return obj.(*kotsv1beta1.Config)
	}

	return nil
}

// getTemplatingKotsKinds returns kots kinds that are necessary/used to populate the contexts of the template builder
func getTemplatingKotsKinds(u *upstreamtypes.Upstream) (*kotsutil.KotsKinds, error) {
	kotsKinds := &kotsutil.KotsKinds{}

	for _, file := range u.Files {
		docs := util.ConvertToSingleDocs(file.Content)
		for _, doc := range docs {
			document := &Document{}
			if err := yaml.Unmarshal(doc, document); err != nil {
				continue
			}

			decode := scheme.Codecs.UniversalDeserializer().Decode
			decoded, gvk, err := decode(doc, nil, nil)
			if err != nil {
				if document.APIVersion == "kots.io/v1beta1" && (document.Kind == "Config" || document.Kind == "License") {
					errMessage := fmt.Sprintf("Failed to decode %s: %v", file.Path, string(doc))
					return nil, errors.Wrap(err, errMessage)
				}
				continue
			}

			switch gvk.String() {
			case "kots.io/v1beta1, Kind=Config":
				kotsKinds.Config = decoded.(*kotsv1beta1.Config)
			case "kots.io/v1beta1, Kind=ConfigValues":
				kotsKinds.ConfigValues = decoded.(*kotsv1beta1.ConfigValues)
			case "kots.io/v1beta1, Kind=Application":
				kotsKinds.KotsApplication = *decoded.(*kotsv1beta1.Application)
			case "kots.io/v1beta1, Kind=License":
				kotsKinds.License = &licensewrapper.LicenseWrapper{V1: decoded.(*kotsv1beta1.License)}
			case "kots.io/v1beta2, Kind=License":
				kotsKinds.License = &licensewrapper.LicenseWrapper{V2: decoded.(*kotsv1beta2.License)}
			case "kots.io/v1beta1, Kind=IdentityConfig":
				kotsKinds.IdentityConfig = decoded.(*kotsv1beta1.IdentityConfig)
			case "kots.io/v1beta1, Kind=Installation":
				kotsKinds.Installation = *decoded.(*kotsv1beta1.Installation)
			}
		}
	}

	return kotsKinds, nil
}

// FindHelmChartArchiveInRelease iterates through all files in the release (upstreamFiles), looking for a helm chart archive
// that matches the chart name and version specified in the kotsHelmChart parameter
func FindHelmChartArchiveInRelease(upstreamFiles []upstreamtypes.UpstreamFile, kotsHelmChart helmchart.HelmChartInterface) ([]byte, error) {
	for _, upstreamFile := range upstreamFiles {
		if !isHelmChart(upstreamFile.Content) {
			continue
		}

		// We treat all .tar.gz archives as helm charts
		chartArchivePath, err := ioutil.TempFile("", "chart")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create temporary file for archive %s", upstreamFile.Path)
		}
		defer os.Remove(chartArchivePath.Name())
		_, err = io.Copy(chartArchivePath, bytes.NewReader(upstreamFile.Content))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to copy archive %s to temporary file", upstreamFile.Path)
		}

		files, err := readTarGz(chartArchivePath.Name())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read chart archive %s", upstreamFile.Path)
		}

		for _, chartFile := range files {
			if chartFile.Path == "Chart.yaml" {
				chartManifest := new(v2chart.Metadata)
				if err := yaml.Unmarshal(chartFile.Content, chartManifest); err != nil {
					return nil, errors.Wrapf(err, "failed to parse Chart.yaml in archive %s", upstreamFile.Path)
				}

				if chartManifest.Name == kotsHelmChart.GetChartName() {
					if chartManifest.Version == kotsHelmChart.GetChartVersion() {
						return upstreamFile.Content, nil
					}
				}
			}
		}
	}

	return nil, errors.Errorf("unable to find helm chart archive for chart name %s, version %s", kotsHelmChart.GetChartName(), kotsHelmChart.GetChartVersion())
}

func isHelmChart(data []byte) bool {
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return false
	}
	gzReader.Close()
	return true
}

func readTarGz(source string) ([]upstreamtypes.UpstreamFile, error) {
	f, err := os.Open(source)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open archive")
	}
	defer f.Close()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gzip reader")
	}

	tarReader := tar.NewReader(gzf)

	upstreamFiles := []upstreamtypes.UpstreamFile{}
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "failed to advance in tar archive")
		}

		name := header.Name

		switch header.Typeflag {
		case tar.TypeReg:
			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(tarReader)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read file from tar archive")
			}
			upstreamFile := upstreamtypes.UpstreamFile{
				Path:    name,
				Content: buf.Bytes(),
			}

			upstreamFiles = append(upstreamFiles, upstreamFile)
		default:
			continue
		}
	}

	// remove any common prefix from all files
	if len(upstreamFiles) > 0 {
		firstFileDir, _ := path.Split(upstreamFiles[0].Path)
		commonPrefix := strings.Split(firstFileDir, string(os.PathSeparator))

		for _, file := range upstreamFiles {
			d, _ := path.Split(file.Path)
			dirs := strings.Split(d, string(os.PathSeparator))

			commonPrefix = util.CommonSlicePrefix(commonPrefix, dirs)

		}

		cleanedUpstreamFiles := []upstreamtypes.UpstreamFile{}
		for _, file := range upstreamFiles {
			d, f := path.Split(file.Path)
			d2 := strings.Split(d, string(os.PathSeparator))

			cleanedUpstreamFile := file
			d2 = d2[len(commonPrefix):]
			cleanedUpstreamFile.Path = path.Join(path.Join(d2...), f)

			cleanedUpstreamFiles = append(cleanedUpstreamFiles, cleanedUpstreamFile)
		}

		upstreamFiles = cleanedUpstreamFiles
	}

	return upstreamFiles, nil
}

func chartArchiveToSparseUpstream(chartArchivePath string) (*upstreamtypes.Upstream, error) {
	files, err := readTarGz(chartArchivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read chart archive")
	}

	upstream := &upstreamtypes.Upstream{
		Type:  "helm",
		Files: files,
	}

	return upstream, nil
}

func removeFileFromUpstream(files []upstreamtypes.UpstreamFile, path string) []upstreamtypes.UpstreamFile {
	for index, file := range files {
		if file.Path == path {
			files[index] = files[len(files)-1]
			files[len(files)-1] = upstreamtypes.UpstreamFile{}
			return files[:len(files)-1]
		}
	}
	return files
}

// helmChartTemplateBuilder resolves defaults only when HelmValue executes.
// Normal rendering remains responsible for parsing and sequencing delimiters.
func helmChartTemplateBuilder(doc []byte, files []upstreamtypes.UpstreamFile, builder template.Builder) template.Builder {
	functions := map[string]interface{}{}
	for name, fn := range builder.Functs {
		functions[name] = fn
	}
	// Chart identity cannot depend on defaults from the chart being identified.
	sourceBuilder := builder
	identityFunctions := map[string]interface{}{}
	for name, fn := range builder.Functs {
		identityFunctions[name] = fn
	}
	identityFunctions["HelmValue"] = func(path string) (string, error) {
		return "", errors.Errorf("HelmValue %q cannot resolve chart name or version before the chart archive is known", path)
	}
	sourceBuilder.Functs = identityFunctions
	resolved := false
	var chartBuilder template.Builder
	var resolveErr error
	functions["HelmValue"] = func(path string) (string, error) {
		if !resolved {
			chartBuilder, resolveErr = resolveHelmChartDefaults(doc, files, sourceBuilder)
			resolved = true
		}
		if resolveErr != nil {
			return "", errors.Wrap(resolveErr, "resolve HelmValue chart defaults")
		}
		value, err := chartBuilder.String(`repl{{ HelmValue ` + strconv.Quote(path) + ` }}`)
		if err != nil {
			return "", errors.Wrapf(err, "read packaged HelmValue %q", path)
		}
		return value, nil
	}
	builder.Functs = functions
	return builder
}

func resolveHelmChartDefaults(doc []byte, files []upstreamtypes.UpstreamFile, builder template.Builder) (template.Builder, error) {
	var identity struct {
		Spec struct {
			Chart struct {
				Name    string `json:"name"`
				Version string `json:"chartVersion"`
			} `json:"chart"`
		} `json:"spec"`
	}
	if err := yaml.Unmarshal(doc, &identity); err != nil {
		return builder, errors.Wrap(err, "failed to parse chart identity")
	}
	name, err := builder.String(identity.Spec.Chart.Name)
	if err != nil {
		return builder, errors.Wrap(err, "failed to render chart name")
	}
	version, err := builder.String(identity.Spec.Chart.Version)
	if err != nil {
		return builder, errors.Wrap(err, "failed to render chart version")
	}
	chart := &kotsv1beta2.HelmChart{}
	chart.Spec.Chart.Name = name
	chart.Spec.Chart.ChartVersion = version
	archive, err := FindHelmChartArchiveInRelease(files, chart)
	if err != nil {
		return builder, errors.Wrapf(err, "failed to resolve archive for chart %s %s", name, version)
	}
	chartBuilder, err := builder.WithHelmChart(archive)
	if err != nil {
		return builder, errors.Wrapf(err, "failed to load defaults for chart %s %s", name, version)
	}
	return chartBuilder, nil
}
