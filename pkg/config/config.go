package config

import (
	"bytes"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func TemplateConfig(log *logger.Logger, configSpecData string, configValuesData string) (string, error) {
	// This function will
	// 1. unmarshal config
	// 2. replace all item values with values that already exist
	// 3. re-marshal it
	// 4. put new config yaml through templating engine
	// This process will re-order items and discard comments, so it should not be saved.

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(configSpecData), nil, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode config data")
	}
	config := obj.(*kotsv1beta1.Config)

	builder := template.Builder{}
	builder.AddCtx(template.StaticCtx{})

	// get template context from config values
	templateContext, err := base.UnmarshalConfigValuesContent([]byte(configValuesData))
	if err != nil {
		log.Error(err)
		templateContext = map[string]template.ItemValue{}
	}

	// add config context
	configCtx, err := builder.NewConfigContext(config.Spec.Groups, templateContext)
	if err != nil {
		return "", errors.Wrap(err, "failed to create config context")
	}

	ApplyValuesToConfig(config, configCtx.ItemValues)
	configDocWithData, err := marshalConfig(config)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal config")
	}

	builder.AddCtx(configCtx)

	rendered, err := builder.RenderTemplate("config", configDocWithData)
	if err != nil {
		return "", errors.Wrap(err, "failed to render config template")
	}

	return rendered, nil
}

func marshalConfig(config *kotsv1beta1.Config) (string, error) {
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var marshalled bytes.Buffer
	if err := s.Encode(config, &marshalled); err != nil {
		return "", errors.Wrap(err, "failed to marshal api role")
	}
	return string(marshalled.Bytes()), nil
}

func ApplyValuesToConfig(config *kotsv1beta1.Config, values map[string]template.ItemValue) {
	for idxG, g := range config.Spec.Groups {
		for idxI, i := range g.Items {
			value, ok := values[i.Name]
			if ok {
				config.Spec.Groups[idxG].Items[idxI].Value = value.ValueStr()
				config.Spec.Groups[idxG].Items[idxI].Default = value.DefaultStr()
			}
			for idxC, c := range i.Items {
				value, ok := values[c.Name]
				if ok {
					config.Spec.Groups[idxG].Items[idxI].Items[idxC].Value = value.ValueStr()
					config.Spec.Groups[idxG].Items[idxI].Items[idxC].Default = value.DefaultStr()
				}
			}
		}
	}
}
