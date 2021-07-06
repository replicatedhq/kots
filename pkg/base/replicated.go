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
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsconfig "github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	stdyaml "gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/yaml"
)

type Document struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
}

func renderReplicated(u *upstreamtypes.Upstream, renderOptions *RenderOptions) (*Base, []Base, error) {
	commonBase := Base{
		Files: []BaseFile{},
		Bases: []Base{},
	}

	builder, itemValues, err := NewConfigContextTemplateBuidler(u, renderOptions)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create new config context template builder")
	}

	config, _, _, _, err := findConfigAndLicense(u, renderOptions.Log)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find config file")
	}

	actualizedConfig, err := kotsconfig.TemplateConfigObjects(config, itemValues, nil, template.LocalRegistry{}, nil, nil, os.Getenv("POD_NAMESPACE"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to template config objects")
	}

	for fileIndex, upstreamFile := range u.Files {
		fmt.Printf("processing upstream file %s\n", upstreamFile.Path)
		if renderOptions.ExcludeKotsKinds {
			// kots kinds are not expected to be valid yaml after builder.RenderTemplate
			// this will prevent errors later from ShouldBeIncludedInBaseKustomization
			newContent := [][]byte{}
			isKotsKind := false
			for _, doc := range convertToSingleDocs(upstreamFile.Content) {
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

		generatedFiles, err := processVariadicConfig(&upstreamFile, actualizedConfig, renderOptions.Log)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process variadic config in file %s", upstreamFile.Path)
		}

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

		if len(generatedFiles) > 0 {
			u.Files = removeFileFromUpstream(u.Files, fileIndex)
			u.Files = append(u.Files, generatedFiles...)
			subBase, err := renderReplicated(u, renderOptions)
			if err != nil {
				return nil, errors.Wrap(err, "failed to render generated variadic files")
			}
			base.Files = addGeneratedFiles(base.Files, subBase.Files)
		}

		if len(generatedFiles) > 0 {
			u.Files = removeFileFromUpstream(u.Files, fileIndex)
			u.Files = append(u.Files, generatedFiles...)
			subBase, err := renderReplicated(u, renderOptions)
			if err != nil {
				return nil, errors.Wrap(err, "failed to render generated variadic files")
			}
			for _, renderedFile := range subBase.Files {
				for _, generatedFile := range generatedFiles {
					if renderedFile.Path == generatedFile.Path {
						base.Files = append(base.Files, renderedFile)
					}
				}
			}
		}
	}

	// render helm charts that were specified
	// we just inject them into u.Files
	kotsHelmCharts, err := findAllKotsHelmCharts(u.Files, *builder, renderOptions.Log)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to find helm charts")
	}

	helmBases := []Base{}
	for _, kotsHelmChart := range kotsHelmCharts {
		helmBase, err := renderReplicatedHelmChart(kotsHelmChart, u.Files, renderOptions, builder)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to render helm chart %s", kotsHelmChart.Name)
		} else if helmBase == nil {
			continue
		}

		renderedHelmBase, err := renderReplicatedHelmBase(u, renderOptions, *helmBase, *builder)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to render helm chart base %s", helmBase.Path)
		}

		renderedHelmBase.Namespace = kotsHelmChart.Spec.Namespace // TODO (ch35027): probably remove this since its already done by "helm template"

		if kotsHelmChart.Spec.UseHelmInstall {
			helmBases = append(helmBases, extractHelmBases(*renderedHelmBase)...)
		} else {
			commonBase.Bases = append(commonBase.Bases, *renderedHelmBase)
		}
	}

	return &commonBase, helmBases, nil
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
	archive, err := findHelmChartArchiveInRelease(upstreamFiles, kotsHelmChart)
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
	helmUpstream.Name = kotsHelmChart.Name

	mergedValues := kotsHelmChart.Spec.Values
	for _, optionalValues := range kotsHelmChart.Spec.OptionalValues {
		parsedBool, err := strconv.ParseBool(optionalValues.When)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse when conditional on optional values")
		}
		if !parsedBool {
			continue
		}
		if optionalValues.RecursiveMerge {
			mergedValues = kotsv1beta1.MergeHelmChartValues(kotsHelmChart.Spec.Values, optionalValues.Values)
		} else {
			for k, v := range optionalValues.Values {
				mergedValues[k] = v
			}
		}
	}

	helmValues, err := kotsHelmChart.Spec.GetHelmValues(mergedValues)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render local values for chart")
	}

	namespace := kotsHelmChart.Spec.Namespace
	if namespace == "" {
		namespace = "repl{{ Namespace}}"
	}

	helmBase, err := RenderHelm(helmUpstream, &RenderOptions{
		SplitMultiDocYAML: true,
		Namespace:         namespace,
		HelmVersion:       kotsHelmChart.Spec.HelmVersion,
		HelmValues:        helmValues,
		ExcludeKotsKinds:  renderOptions.ExcludeKotsKinds,
		Log:               nil,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to render helm chart in upstream")
	}

	helmBase.Path = path.Join("charts", helmUpstream.Name)
	return helmBase, nil
}

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
		Files:           helmBaseFiles,
		AdditionalFiles: helmBase.AdditionalFiles,
		Bases:           helmBaseBases,
		ErrorFiles:      helmBaseYAMLErrorFiles,
	}, nil
}

func upstreamFileToBaseFile(upstreamFile upstreamtypes.UpstreamFile, builder template.Builder, log *logger.CLILogger) (BaseFile, error) {
	rendered, err := builder.RenderTemplate(upstreamFile.Path, string(upstreamFile.Content))
	if err != nil {
		log.Error(errors.Errorf("Failed to render file %s. Contents are %s", upstreamFile.Path, upstreamFile.Content))
		return BaseFile{}, errors.Wrap(err, "failed to render file template")
	}

	return BaseFile{
		Path:    upstreamFile.Path,
		Content: []byte(rendered),
	}, nil
}

func findAllKotsHelmCharts(upstreamFiles []upstreamtypes.UpstreamFile, builder template.Builder, log *logger.CLILogger) ([]*kotsv1beta1.HelmChart, error) {
	kotsHelmCharts := []*kotsv1beta1.HelmChart{}
	for _, upstreamFile := range upstreamFiles {
		if !isHelmChartKind(upstreamFile.Content) {
			continue
		}

		baseFile, err := upstreamFileToBaseFile(upstreamFile, builder, log)
		if err != nil {
			continue
		}

		helmChart, err := parseHelmChart(baseFile.Content)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rendered HelmChart %s", baseFile.Path)
		}

		kotsHelmCharts = append(kotsHelmCharts, helmChart)
	}

	return kotsHelmCharts, nil
}

func UnmarshalLicenseContent(content []byte, log *logger.CLILogger) *kotsv1beta1.License {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		log.Info("Failed to parse file while looking for license: %v", err)
		return nil
	}

	if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "License" {
		return obj.(*kotsv1beta1.License)
	}

	return nil
}

func parseHelmChart(content []byte) (*kotsv1beta1.HelmChart, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode chart")
	}

	if gvk.Group == "kots.io" {
		if gvk.Version == "v1beta1" {
			if gvk.Kind == "HelmChart" {
				return obj.(*kotsv1beta1.HelmChart), nil
			}
		}
	}

	return nil, errors.Errorf("not a HelmChart GVK: %s", gvk.String())
}

func isHelmChartKind(content []byte) bool {
	gvk := OverlySimpleGVK{}

	if err := stdyaml.Unmarshal(content, &gvk); err != nil {
		return false
	}

	if gvk.APIVersion == "kots.io/v1beta1" && gvk.Kind == "HelmChart" {
		return true
	}
	return false
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

func findConfigAndLicense(u *upstreamtypes.Upstream, log *logger.CLILogger) (*kotsv1beta1.Config, *kotsv1beta1.ConfigValues, *kotsv1beta1.IdentityConfig, *kotsv1beta1.License, error) {
	var config *kotsv1beta1.Config
	var values *kotsv1beta1.ConfigValues
	var identityConfig *kotsv1beta1.IdentityConfig
	var license *kotsv1beta1.License

	for _, file := range u.Files {
		document := &Document{}
		if err := yaml.Unmarshal(file.Content, document); err != nil {
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode(file.Content, nil, nil)
		if err != nil {
			if document.APIVersion == "kots.io/v1beta1" && (document.Kind == "Config" || document.Kind == "License") {
				errMessage := fmt.Sprintf("Failed to decode %s", file.Path)
				return nil, nil, nil, nil, errors.Wrap(err, errMessage)
			}
			continue
		}

		if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Config" {
			config = obj.(*kotsv1beta1.Config)
		} else if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "ConfigValues" {
			values = obj.(*kotsv1beta1.ConfigValues)
		} else if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "IdentityConfig" {
			identityConfig = obj.(*kotsv1beta1.IdentityConfig)
		} else if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "License" {
			license = obj.(*kotsv1beta1.License)
		}
	}

	return config, values, identityConfig, license, nil
}

// findHelmChartArchiveInRelease iterates through all files in the release (upstreamFiles), looking for a helm chart archive
// that matches the chart name and version specified in the kotsHelmChart parameter
func findHelmChartArchiveInRelease(upstreamFiles []upstreamtypes.UpstreamFile, kotsHelmChart *kotsv1beta1.HelmChart) ([]byte, error) {
	for _, upstreamFile := range upstreamFiles {
		if !isHelmChart(upstreamFile.Content) {
			continue
		}

		// We treat all .tar.gz archives as helm charts
		chartArchivePath, err := ioutil.TempFile("", "chart")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create temp file for chart archive path")
		}
		defer os.Remove(chartArchivePath.Name())
		_, err = io.Copy(chartArchivePath, bytes.NewReader(upstreamFile.Content))
		if err != nil {
			return nil, errors.Wrap(err, "failed to copy maybe chart to tmp file")
		}

		files, err := readTarGz(chartArchivePath.Name())
		if err != nil {
			return nil, errors.Wrap(err, "failed to read chart archive")
		}

		for _, chartFile := range files {
			if chartFile.Path == "Chart.yaml" {
				chartManifest := new(chart.Metadata)
				if err := yaml.Unmarshal(chartFile.Content, chartManifest); err != nil {
					return nil, errors.Wrap(err, "failed to unmarshal chart yaml")
				}

				if chartManifest.Name == kotsHelmChart.Spec.Chart.Name {
					if chartManifest.Version == kotsHelmChart.Spec.Chart.ChartVersion {
						return upstreamFile.Content, nil
					}
				}
			}
		}
	}

	return nil, errors.Errorf("unable to find helm chart archive for chart name %s, version %s", kotsHelmChart.Spec.Chart.Name, kotsHelmChart.Spec.Chart.ChartVersion)
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

func removeFileFromUpstream(files []upstreamtypes.UpstreamFile, index int) []upstreamtypes.UpstreamFile {
	files[index] = files[len(files)-1]
	return files[:len(files)-1]
}

func addGeneratedFiles(files []BaseFile, subFiles []BaseFile) []BaseFile {
	for _, subFile := range subFiles {
		for _, file := range files {
			if subFile.Path == file.Path {
				continue
			}
		}
		fmt.Printf("appending file %s\n\n", subFile.Path)
		files = append(files, subFile)
	}

	return files
}
