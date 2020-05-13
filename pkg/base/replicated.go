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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/yaml"
)

type Document struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
}

func renderReplicated(u *upstreamtypes.Upstream, renderOptions *RenderOptions) (*Base, error) {
	config, configValues, license, err := findConfigAndLicense(u, renderOptions.Log)
	if err != nil {
		return nil, err
	}

	var templateContext map[string]template.ItemValue
	if configValues != nil {
		ctx := map[string]template.ItemValue{}
		for k, v := range configValues.Spec.Values {
			ctx[k] = template.ItemValue{
				Value:   v.Value,
				Default: v.Default,
			}
		}
		templateContext = ctx
	} else {
		templateContext = map[string]template.ItemValue{}
	}

	var cipher *crypto.AESCipher
	if u.EncryptionKey != "" {
		c, err := crypto.AESCipherFromString(u.EncryptionKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create cipher")
		}
		cipher = c
	}

	base := Base{
		Files: []BaseFile{},
		Bases: []Base{},
	}

	configGroups := []kotsv1beta1.ConfigGroup{}
	if config != nil {
		configGroups = config.Spec.Groups
	}

	localRegistry := template.LocalRegistry{
		Host:      renderOptions.LocalRegistryHost,
		Namespace: renderOptions.LocalRegistryNamespace,
		Username:  renderOptions.LocalRegistryUsername,
		Password:  renderOptions.LocalRegistryPassword,
	}

	builder, _, err := template.NewBuilder(configGroups, templateContext, localRegistry, cipher, license)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create config context")
	}

	for _, upstreamFile := range u.Files {
		baseFile, err := upstreamFileToBaseFile(upstreamFile, builder, renderOptions.Log)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert upstream file %s to base", upstreamFile.Path)
		}

		baseFiles := convertToSingleDocs([]BaseFile{baseFile})
		for _, f := range baseFiles {
			include, err := f.ShouldBeIncludedInBaseKustomization(renderOptions.ExcludeKotsKinds)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to check if base file %s should be included", f.Path)
			}
			if include {
				base.Files = append(base.Files, f)
			}
		}
	}

	// render helm charts that were specified
	// we just inject them into u.Files
	kotsHelmCharts := findAllKotsHelmCharts(u.Files)
	for _, kotsHelmChart := range kotsHelmCharts {
		if kotsHelmChart.Spec.Exclude != "" {
			renderedExclude, err := builder.RenderTemplate(kotsHelmChart.Name, kotsHelmChart.Spec.Exclude)
			if err != nil {
				renderOptions.Log.Error(errors.Errorf("Failed to render helm chart exclude %s", kotsHelmChart.Spec.Exclude))
				return nil, errors.Wrap(err, "failed to render kots helm chart exclude")
			}

			parsedBool, err := strconv.ParseBool(renderedExclude)
			if err != nil {
				renderOptions.Log.Error(errors.Errorf("Kots.io/v1beta1 HelmChart rendered exclude is not parseable as bool, value = %s, filename = %s. Not excluding chart.", kotsHelmChart.Spec.Exclude, u.Name))
				return nil, errors.Wrap(err, "failed to parse helm chart exclude")
			}

			if parsedBool {
				continue
			}
		}

		// Include this chart
		archive, err := findHelmChartArchiveInRelease(u.Files, kotsHelmChart)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find helm chart archive in release")
		}

		tmpFile, err := ioutil.TempFile("", "kots")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create temp file")
		}
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
			renderedWhen, err := builder.RenderTemplate(kotsHelmChart.Name, optionalValues.When)
			if err != nil {
				return nil, errors.Wrap(err, "failed to render when from conditional on optional value")
			}
			parsedBool, err := strconv.ParseBool(renderedWhen)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse when conditional on optional values")
			}
			if !parsedBool {
				continue
			}

			for k, v := range optionalValues.Values {
				mergedValues[k] = v
			}
		}

		localValues, err := kotsHelmChart.Spec.RenderValues(mergedValues, func(s2 string) (s string, err error) {
			return builder.RenderTemplate(s2, s2)
		})
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
			HelmOptions:       localValues,
			ExcludeKotsKinds:  renderOptions.ExcludeKotsKinds,
			Log:               nil,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to render helm chart in upstream")
		}

		helmBaseFiles := []BaseFile{}
		for _, helmBaseFile := range helmBase.Files {
			filePath := filepath.Join("charts", kotsHelmChart.Name, helmBaseFile.Path)

			// this is a little bit of an abuse of the next function
			include, err := helmBaseFile.ShouldBeIncludedInBaseKustomization(false)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to determine if file %s should be included in base", filePath)
			}

			if !include {
				continue
			}

			upstreamFile := upstreamtypes.UpstreamFile{
				Path:    filePath,
				Content: helmBaseFile.Content,
			}

			u.Files = append(u.Files, upstreamFile)

			baseFile, err := upstreamFileToBaseFile(upstreamFile, builder, renderOptions.Log)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to convert upstream file %s to base", filePath)
			}

			helmBaseFiles = append(helmBaseFiles, baseFile)
		}

		if kotsHelmChart.Spec.Namespace != "" {
			base.Bases = append(base.Bases, Base{
				Path:      filepath.Join("charts", kotsHelmChart.Name),
				Namespace: kotsHelmChart.Spec.Namespace,
				Files:     helmBaseFiles,
			})
		} else {
			base.Files = append(base.Files, helmBaseFiles...)
		}
	}

	return &base, nil
}

func upstreamFileToBaseFile(upstreamFile types.UpstreamFile, builder template.Builder, log *logger.Logger) (BaseFile, error) {
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

func findAllKotsHelmCharts(upstreamFiles []upstreamtypes.UpstreamFile) []*kotsv1beta1.HelmChart {
	kotsHelmCharts := []*kotsv1beta1.HelmChart{}
	for _, upstreamFile := range upstreamFiles {
		kotsHelmChart := tryParsingAsHelmChartGVK(upstreamFile.Content)
		if kotsHelmChart != nil {
			kotsHelmCharts = append(kotsHelmCharts, kotsHelmChart)
		}
	}

	return kotsHelmCharts
}

func UnmarshalLicenseContent(content []byte, log *logger.Logger) *kotsv1beta1.License {
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

func UnmarshalConfigValuesContent(content []byte) (map[string]template.ItemValue, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode values")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "ConfigValues" {
		return nil, errors.New("not a configvalues object")
	}

	values := obj.(*kotsv1beta1.ConfigValues)

	ctx := map[string]template.ItemValue{}
	for k, v := range values.Spec.Values {
		ctx[k] = template.ItemValue{
			Value:   v.Value,
			Default: v.Default,
		}
	}

	return ctx, nil
}

func tryParsingAsHelmChartGVK(content []byte) *kotsv1beta1.HelmChart {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil
	}

	if gvk.Group == "kots.io" {
		if gvk.Version == "v1beta1" {
			if gvk.Kind == "HelmChart" {
				return obj.(*kotsv1beta1.HelmChart)
			}
		}
	}

	return nil
}

func tryGetConfigFromFileContent(content []byte, log *logger.Logger) *kotsv1beta1.Config {
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

func findConfigAndLicense(u *upstreamtypes.Upstream, log *logger.Logger) (*kotsv1beta1.Config, *kotsv1beta1.ConfigValues, *kotsv1beta1.License, error) {
	var config *kotsv1beta1.Config
	var values *kotsv1beta1.ConfigValues
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
				return nil, nil, nil, errors.Wrap(err, errMessage)
			}
			continue
		}

		if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Config" {
			config = obj.(*kotsv1beta1.Config)
		} else if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "ConfigValues" {
			values = obj.(*kotsv1beta1.ConfigValues)
		} else if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "License" {
			license = obj.(*kotsv1beta1.License)
		}
	}

	return config, values, license, nil
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

	upstreamFiles := []types.UpstreamFile{}
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
			upstreamFile := types.UpstreamFile{
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

		cleanedUpstreamFiles := []types.UpstreamFile{}
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
