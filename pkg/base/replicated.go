package base

import (
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/upstream"
	"k8s.io/client-go/kubernetes/scheme"
)

func renderReplicated(u *upstream.Upstream, renderOptions *RenderOptions) (*Base, error) {
	// Find the config for the config groups
	var config *kotsv1beta1.Config
	for _, upstreamFile := range u.Files {
		maybeConfig := tryGetConfigFromFileContent(upstreamFile.Content)
		if maybeConfig != nil {
			config = maybeConfig
		}
	}

	// Find the values from the context
	var templateContext map[string]interface{}
	for _, c := range u.Files {
		if c.Path == "userdata/config.yaml" {
			ctx, err := UnmarshalConfigValuesContent(c.Content)
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

	base := Base{
		Files: baseFiles,
	}

	return &base, nil
}

func UnmarshalConfigValuesContent(content []byte) (map[string]interface{}, error) {
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
