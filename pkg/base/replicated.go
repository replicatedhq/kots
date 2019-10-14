package base

import (
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/upstream"
	"k8s.io/client-go/kubernetes/scheme"
)

func renderReplicated(u *upstream.Upstream, renderOptions *RenderOptions) (*Base, error) {
	// Find the config for the config groups and the license
	var config *kotsv1beta1.Config
	var license *kotsv1beta1.License
	var licenseData []byte
	for _, upstreamFile := range u.Files {
		maybeConfig := tryGetConfigFromFileContent(upstreamFile.Content)
		if maybeConfig != nil {
			config = maybeConfig
		}

		maybeLicense := tryGetLicenseFromFileContent(upstreamFile.Content)
		if maybeLicense != nil {
			license = maybeLicense
			licenseData = upstreamFile.Content
		}
	}

	// Find the values from the context
	var templateContext map[string]interface{}
	for _, c := range u.Files {
		if c.Path == "userdata/config.yaml" {
			ctx, err := unmarshalConfigValuesContent(c.Content)
			if err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal config values content")
			}

			templateContext = ctx
		}
	}

	baseFiles := []BaseFile{}

	builder := template.Builder{}
	builder.AddCtx(template.StaticCtx{})

	if config != nil {
		configCtx, err := builder.NewConfigContext(config.Spec.Groups, templateContext)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create config context")
		}
		builder.AddCtx(configCtx)
	}

	for _, upstreamFile := range u.Files {
		rendered, err := builder.RenderTemplate(upstreamFile.Path, string(upstreamFile.Content))
		if err != nil {
			return nil, errors.Wrap(err, "failed to render template")
		}

		baseFile := BaseFile{
			Path:    upstreamFile.Path,
			Content: []byte(rendered),
		}

		baseFiles = append(baseFiles, baseFile)
	}

	// add titled
	titledDocs, err := kotsadm.GetTitledYAML(licenseData, license)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get titled docs")
	}
	for titledFilename, titledManifest := range titledDocs {
		baseFile := BaseFile{
			Path:    titledFilename,
			Content: titledManifest,
		}

		baseFiles = append(baseFiles, baseFile)
	}

	base := Base{
		Files: baseFiles,
	}

	return &base, nil
}

func unmarshalConfigValuesContent(content []byte) (map[string]interface{}, error) {
	kotsscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode values")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "ConfigValues" {
		return nil, errors.New("not a configvalues object")
	}

	values := obj.(*kotsv1beta1.ConfigValues)

	ctx := map[string]interface{}{}
	for k, v := range values.Spec.Values {
		ctx[k] = v
	}

	return ctx, nil
}

func tryGetConfigFromFileContent(content []byte) *kotsv1beta1.Config {
	kotsscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil
	}

	if gvk.Group == "kots.io" {
		if gvk.Version == "v1beta1" {
			if gvk.Kind == "Config" {
				return obj.(*kotsv1beta1.Config)
			}
		}
	}

	return nil
}

func tryGetLicenseFromFileContent(content []byte) *kotsv1beta1.License {
	kotsscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil
	}

	if gvk.Group == "kots.io" {
		if gvk.Version == "v1beta1" {
			if gvk.Kind == "License" {
				return obj.(*kotsv1beta1.License)
			}
		}
	}

	return nil
}
