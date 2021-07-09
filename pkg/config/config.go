package config

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/multitype"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/util"
	yaml "github.com/replicatedhq/yaml/v3"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func TemplateConfig(log *logger.CLILogger, configSpecData string, configValuesData string, licenseData string, identityConfigData string, localRegistry template.LocalRegistry, namespace string) (string, error) {
	return templateConfig(log, configSpecData, configValuesData, licenseData, identityConfigData, localRegistry, namespace, MarshalConfig)
}

func TemplateConfigObjects(configSpec *kotsv1beta1.Config, configValues map[string]template.ItemValue, license *kotsv1beta1.License, localRegistry template.LocalRegistry, versionInfo *template.VersionInfo, identityconfig *kotsv1beta1.IdentityConfig, namespace string) (*kotsv1beta1.Config, error) {
	templatedString, err := templateConfigObjects(configSpec, configValues, license, localRegistry, versionInfo, identityconfig, namespace, MarshalConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to template config")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode([]byte(templatedString), nil, nil) // TODO fix decode of boolstrings
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode config data")
	}
	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Config" {
		return nil, errors.Errorf("expected Config, but found %s/%s/%s", gvk.Group, gvk.Version, gvk.Kind)
	}
	config := obj.(*kotsv1beta1.Config)
	return config, nil
}

func templateConfigObjects(configSpec *kotsv1beta1.Config, configValues map[string]template.ItemValue, license *kotsv1beta1.License, localRegistry template.LocalRegistry, versionInfo *template.VersionInfo, identityconfig *kotsv1beta1.IdentityConfig, namespace string, marshalFunc func(config *kotsv1beta1.Config) (string, error)) (string, error) {
	builderOptions := template.BuilderOptions{
		ConfigGroups:   configSpec.Spec.Groups,
		ExistingValues: configValues,
		LocalRegistry:  localRegistry,
		Cipher:         nil,
		License:        license,
		VersionInfo:    versionInfo,
		IdentityConfig: identityconfig,
		Namespace:      namespace,
	}
	builder, configVals, err := template.NewBuilder(builderOptions)
	if err != nil {
		return "", errors.Wrap(err, "failed to create config context")
	}

	ApplyValuesToConfig(configSpec, configVals)
	configDocWithData, err := marshalFunc(configSpec)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal config")
	}

	rendered, err := builder.RenderTemplate("config", string(configDocWithData))
	if err != nil {
		return "", errors.Wrap(err, "failed to render config template")
	}

	return rendered, nil
}

func templateConfig(log *logger.CLILogger, configSpecData string, configValuesData string, licenseData string, identityConfigData string, localRegistry template.LocalRegistry, namespace string, marshalFunc func(config *kotsv1beta1.Config) (string, error)) (string, error) {
	// This function will
	// 1. unmarshal config
	// 2. replace all item values with values that already exist
	// 3. evaluate the dependency graph for config values (template function chaining)
	// 4. re-marshal it (with an unlimited line length)
	// 5. put new config yaml through templating engine
	// This process will re-order items and discard comments, so it should not be saved.
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode([]byte(licenseData), nil, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode license data")
	}
	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "License" {
		return "", errors.Errorf("expected License, but found %s/%s/%s", gvk.Group, gvk.Version, gvk.Kind)
	}
	license := obj.(*kotsv1beta1.License)

	obj, gvk, err = decode([]byte(configSpecData), nil, nil) // TODO fix decode of boolstrings
	if err != nil {
		return "", errors.Wrap(err, "failed to decode config data")
	}
	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Config" {
		return "", errors.Errorf("expected Config, but found %s/%s/%s", gvk.Group, gvk.Version, gvk.Kind)
	}
	config := obj.(*kotsv1beta1.Config)

	// get template context from config values
	templateContext, err := UnmarshalConfigValuesContent([]byte(configValuesData))
	if err != nil {
		log.Error(err)
		templateContext = map[string]template.ItemValue{}
	}

	var identityConfig *kotsv1beta1.IdentityConfig
	if identityConfigData != "" {
		obj, gvk, err = decode([]byte(identityConfigData), nil, nil)
		if err != nil {
			return "", errors.Wrap(err, "failed to decode config data")
		}
		if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "IdentityConfig" {
			return "", errors.Errorf("expected IdentityConfig, but found %s/%s/%s", gvk.Group, gvk.Version, gvk.Kind)
		}
		identityConfig = obj.(*kotsv1beta1.IdentityConfig)
	}

	return templateConfigObjects(config, templateContext, license, localRegistry, &template.VersionInfo{}, identityConfig, namespace, marshalFunc)
}

func ApplyValuesToConfig(config *kotsv1beta1.Config, values map[string]template.ItemValue) {
	for idxG, g := range config.Spec.Groups {
		for idxI, i := range g.Items {
			// if the item is repeatable
			if i.Repeatable {
				if config.Spec.Groups[idxG].Items[idxI].ValuesByGroup == nil {
					// initialize the appropriate maps
					config.Spec.Groups[idxG].Items[idxI].ValuesByGroup = map[string]kotsv1beta1.GroupValues{}
				}
				if config.Spec.Groups[idxG].Items[idxI].CountByGroup == nil {
					config.Spec.Groups[idxG].Items[idxI].CountByGroup = map[string]int{}
				}
				if config.Spec.Groups[idxG].Items[idxI].ValuesByGroup[g.Name] == nil {
					config.Spec.Groups[idxG].Items[idxI].ValuesByGroup[g.Name] = map[string]interface{}{}
					config.Spec.Groups[idxG].Items[idxI].CountByGroup[g.Name] = 0
				}
				for fieldName, item := range values {
					if item.RepeatableItem == i.Name {
						config.Spec.Groups[idxG].Items[idxI].ValuesByGroup[g.Name][fieldName] = item.Value
					}
				}
				for variadicGroup, groupValues := range config.Spec.Groups[idxG].Items[idxI].ValuesByGroup {
					config.Spec.Groups[idxG].Items[idxI].CountByGroup[variadicGroup] = len(groupValues)
				}
				CreateVariadicValues(&config.Spec.Groups[idxG].Items[idxI], g.Name)
			}
			value, ok := values[i.Name]
			if ok {
				config.Spec.Groups[idxG].Items[idxI].Value = multitype.FromString(value.ValueStr())
				config.Spec.Groups[idxG].Items[idxI].Default = multitype.FromString(value.DefaultStr())

				if value.Filename != "" {
					config.Spec.Groups[idxG].Items[idxI].Filename = value.Filename
				}
			}
			for idxC, c := range i.Items {
				value, ok := values[c.Name]
				if ok {
					config.Spec.Groups[idxG].Items[idxI].Items[idxC].Value = multitype.FromString(value.ValueStr())
					config.Spec.Groups[idxG].Items[idxI].Items[idxC].Default = multitype.FromString(value.DefaultStr())

					if value.Filename != "" {
						config.Spec.Groups[idxG].Items[idxI].Filename = value.Filename
					}
				}
			}
		}
	}
}

// MarshalConfig runs the same code path as the k8s json->yaml serializer, but uses a different yaml library for those parts
// first, the object is marshalled to json
// second, the json is unmarshalled to an object as yaml
// third, the object is marshalled as yaml
func MarshalConfig(config *kotsv1beta1.Config) (string, error) {
	s := json.NewSerializerWithOptions(
		json.DefaultMetaFactory,
		scheme.Scheme,
		scheme.Scheme,
		json.SerializerOptions{Yaml: false, Pretty: true, Strict: false},
	)

	var marshalledJSON bytes.Buffer
	if err := s.Encode(config, &marshalledJSON); err != nil {
		return "", errors.Wrap(err, "failed to marshal config as json")
	}

	var unmarshalledYAML interface{}
	if err := yaml.Unmarshal(marshalledJSON.Bytes(), &unmarshalledYAML); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal config as yaml")
	}

	marshalledYAML, err := util.MarshalIndent(2, unmarshalledYAML)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal config as yaml")
	}

	return string(marshalledYAML), nil
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
			Value:          v.Value,
			Default:        v.Default,
			RepeatableItem: v.RepeatableItem,
		}
	}

	return ctx, nil
}

func CreateVariadicValues(item *kotsv1beta1.ConfigItem, groupName string) {
	if item.ValuesByGroup == nil {
		item.ValuesByGroup = map[string]kotsv1beta1.GroupValues{}
	}
	if item.CountByGroup == nil {
		item.CountByGroup = map[string]int{}
	}

	if item.MinimumCount != 0 && (item.CountByGroup[groupName] < item.MinimumCount) {
		item.CountByGroup[groupName] = item.MinimumCount
	} else if item.CountByGroup[groupName] == 0 {
		item.CountByGroup[groupName] = 1
	}

	for len(item.ValuesByGroup[groupName]) < item.CountByGroup[groupName] {
		itemValue := template.ItemValue{
			Value:          "",
			RepeatableItem: item.Name,
		}
		shortUUID := strings.Split(uuid.New().String(), "-")[0]
		variadicName := fmt.Sprintf("%s-%s", item.Name, shortUUID)
		item.ValuesByGroup[groupName][variadicName] = itemValue
	}
}
