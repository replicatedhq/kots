package base

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"k8s.io/client-go/kubernetes/scheme"
)

func renderReplicated(u *upstreamtypes.Upstream, renderOptions *RenderOptions) (*Base, error) {
	config, configValues, license := findConfig(u, renderOptions.Log)

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

	baseFiles := []BaseFile{}

	builder := template.Builder{}
	builder.AddCtx(template.StaticCtx{})

	if config != nil {
		configCtx, err := builder.NewConfigContext(config.Spec.Groups, templateContext, cipher)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create config context")
		}
		builder.AddCtx(configCtx)
	}

	if license != nil {
		licenseCtx := template.LicenseCtx{
			License: license,
		}
		builder.AddCtx(licenseCtx)
	}

	// loop through all files once, looking for helm charts so we can exclude them, if they "when" out
	excludedPathPrefixes := []string{}
	for _, upstreamFile := range u.Files {
		kotsHelmChart := tryParsingAsHelmChartGVK(upstreamFile.Content)

		if kotsHelmChart == nil {
			continue
		}

		rendered, err := builder.RenderTemplate(upstreamFile.Path, string(upstreamFile.Content))
		if err != nil {
			renderOptions.Log.Error(errors.Errorf("Failed to render helm chart manifest %s. Contents are %s", upstreamFile.Path, upstreamFile.Content))
			return nil, errors.Wrap(err, "failed to render kots helm chart kind")
		}

		kotsHelmChart = tryParsingAsHelmChartGVK([]byte(rendered))
		if kotsHelmChart == nil {
			renderOptions.Log.Error(errors.Errorf("Kots.io/v1beta1 HelmChart rendered to an unparseable kind. Filename %s. Using the underendered version of this file.", upstreamFile.Path))
			kotsHelmChart = tryParsingAsHelmChartGVK(upstreamFile.Content)
		}

		if kotsHelmChart.Spec.Exclude != "" {
			parsedBool, err := strconv.ParseBool(kotsHelmChart.Spec.Exclude)
			if err != nil {
				renderOptions.Log.Error(errors.Errorf("Kots.io/v1beta1 HelmChart rendered exclude is not parseable as bool, value = %s, filename = %s. Not excluding chart.", kotsHelmChart.Spec.Exclude, upstreamFile.Path))
			}

			if parsedBool {
				excludedPathPrefixes = append(excludedPathPrefixes, filepath.Join("charts", kotsHelmChart.Name)+string(os.PathSeparator))
			}
		}
	}

	for _, upstreamFile := range u.Files {
		// most apps will have 0-<very small int> of helm charts
		excludeFile := false
		for _, excludedPathPrefix := range excludedPathPrefixes {
			if strings.HasPrefix(upstreamFile.Path, excludedPathPrefix) {
				excludeFile = true
				break
			}
		}
		if excludeFile {
			continue
		}

		rendered, err := builder.RenderTemplate(upstreamFile.Path, string(upstreamFile.Content))
		if err != nil {
			renderOptions.Log.Error(errors.Errorf("Failed to render file %s. Contents are %s", upstreamFile.Path, upstreamFile.Content))
			return nil, errors.Wrap(err, "failed to render file template")
		}

		baseFile := BaseFile{
			Path:    upstreamFile.Path,
			Content: []byte(rendered),
		}

		baseFiles = append(baseFiles, baseFile)
	}

	base := Base{
		Files: baseFiles,
	}

	return &base, nil
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

func findConfig(u *upstreamtypes.Upstream, log *logger.Logger) (*kotsv1beta1.Config, *kotsv1beta1.ConfigValues, *kotsv1beta1.License) {
	var config *kotsv1beta1.Config
	var values *kotsv1beta1.ConfigValues
	var license *kotsv1beta1.License

	for _, file := range u.Files {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode(file.Content, nil, nil)
		if err != nil {
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
	return config, values, license
}
