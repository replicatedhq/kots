package config

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/template"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/multitype"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
	goyaml "go.yaml.in/yaml/v3"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

// Regexps for decoding Unicode escape sequences introduced by yaml.v2 marshalling.
// yaml.v2 escapes non-BMP characters (codepoints > U+FFFF) as \UXXXXXXXX because its
// isPrintable() excludes the 0x10000-0x10FFFF range. These escape sequences break
// Go's text/template parser when they appear in config fields alongside repl{{}} expressions.
var (
	reUnicodeEscape8 = regexp.MustCompile(`\\U([0-9A-Fa-f]{8})`)
	reSurrogatePair  = regexp.MustCompile(`\\u([Dd][89ABab][0-9A-Fa-f]{2})\\u([Dd][C-Fc-f][0-9A-Fa-f]{2})`)
	reUnicodeEscape4 = regexp.MustCompile(`\\u([0-9A-Fa-f]{4})`)
)

func TemplateConfigObjects(configSpec *kotsv1beta1.Config, configValues map[string]template.ItemValue, license *licensewrapper.LicenseWrapper, app *kotsv1beta1.Application, localRegistry registrytypes.RegistrySettings, versionInfo *template.VersionInfo, appInfo *template.ApplicationInfo, identityconfig *kotsv1beta1.IdentityConfig, namespace string, decryptValues bool) (*kotsv1beta1.Config, error) {
	templatedString, err := templateConfigObjects(configSpec, configValues, license, app, localRegistry, versionInfo, appInfo, identityconfig, namespace, decryptValues, MarshalConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to template config")
	}

	if len(templatedString) == 0 {
		return nil, nil
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

func templateConfigObjects(configSpec *kotsv1beta1.Config, configValues map[string]template.ItemValue, license *licensewrapper.LicenseWrapper, app *kotsv1beta1.Application, localRegistry registrytypes.RegistrySettings, versionInfo *template.VersionInfo, appInfo *template.ApplicationInfo, identityconfig *kotsv1beta1.IdentityConfig, namespace string, decryptValues bool, marshalFunc func(config *kotsv1beta1.Config) (string, error)) (string, error) {
	if configSpec == nil {
		return "", nil
	}

	builderOptions := template.BuilderOptions{
		ConfigGroups:    configSpec.Spec.Groups,
		ExistingValues:  configValues,
		LocalRegistry:   localRegistry,
		License:         license,
		Application:     app,
		VersionInfo:     versionInfo,
		ApplicationInfo: appInfo,
		IdentityConfig:  identityconfig,
		Namespace:       namespace,
		DecryptValues:   decryptValues,
	}

	builder, configVals, err := template.NewBuilder(builderOptions)
	if err != nil {
		return "", errors.Wrap(err, "failed to create config context")
	}

	actualizedConfig := ApplyValuesToConfig(configSpec, configVals)
	configDocWithData, err := marshalFunc(actualizedConfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal config")
	}

	rendered, err := builder.RenderTemplate("config", string(configDocWithData))
	if err != nil {
		return "", errors.Wrap(err, "failed to render config template")
	}

	return rendered, nil
}

func ApplyValuesToConfig(config *kotsv1beta1.Config, values map[string]template.ItemValue) *kotsv1beta1.Config {
	if config == nil {
		return nil
	}

	configInstance := config.DeepCopy()

	for idxG, g := range configInstance.Spec.Groups {
		for idxI, i := range g.Items {
			// if the item is repeatable
			if i.Repeatable {
				if configInstance.Spec.Groups[idxG].Items[idxI].ValuesByGroup == nil {
					// initialize the appropriate maps
					configInstance.Spec.Groups[idxG].Items[idxI].ValuesByGroup = map[string]kotsv1beta1.GroupValues{}
				}
				if configInstance.Spec.Groups[idxG].Items[idxI].CountByGroup == nil {
					configInstance.Spec.Groups[idxG].Items[idxI].CountByGroup = map[string]int{}
				}
				if configInstance.Spec.Groups[idxG].Items[idxI].ValuesByGroup[g.Name] == nil {
					configInstance.Spec.Groups[idxG].Items[idxI].ValuesByGroup[g.Name] = map[string]string{}
					configInstance.Spec.Groups[idxG].Items[idxI].CountByGroup[g.Name] = 0
				}
				for fieldName, item := range values {
					if item.RepeatableItem == i.Name {
						configInstance.Spec.Groups[idxG].Items[idxI].ValuesByGroup[g.Name][fieldName] = fmt.Sprintf("%s", item.Value)
					}
				}
				for variadicGroup, groupValues := range configInstance.Spec.Groups[idxG].Items[idxI].ValuesByGroup {
					configInstance.Spec.Groups[idxG].Items[idxI].CountByGroup[variadicGroup] = len(groupValues)
				}
				CreateVariadicValues(&configInstance.Spec.Groups[idxG].Items[idxI], g.Name)
			}
			value, ok := values[i.Name]
			if ok {
				configInstance.Spec.Groups[idxG].Items[idxI].Value = multitype.FromString(value.ValueStr())
				configInstance.Spec.Groups[idxG].Items[idxI].Default = multitype.FromString(value.DefaultStr())

				if value.Filename != "" {
					configInstance.Spec.Groups[idxG].Items[idxI].Filename = value.Filename
				}
			}
			for idxC, c := range i.Items {
				value, ok := values[c.Name]
				if ok {
					configInstance.Spec.Groups[idxG].Items[idxI].Items[idxC].Value = multitype.FromString(value.ValueStr())
					configInstance.Spec.Groups[idxG].Items[idxI].Items[idxC].Default = multitype.FromString(value.DefaultStr())

					if value.Filename != "" {
						configInstance.Spec.Groups[idxG].Items[idxI].Filename = value.Filename
					}
				}
			}
		}
	}

	return configInstance
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
		json.SerializerOptions{Yaml: true, Pretty: false, Strict: false},
	)

	var marshalledYAML bytes.Buffer
	if err := s.Encode(config, &marshalledYAML); err != nil {
		return "", errors.Wrap(err, "failed to marshal config as json")
	}

	b, err := kotsutil.FixUpYAML(marshalledYAML.Bytes())
	if err != nil {
		return "", errors.Wrap(err, "failed to fix up yaml")
	}

	normalized, err := normalizeTemplateStyles(b)
	if err != nil {
		return "", errors.Wrap(err, "failed to normalize template styles")
	}

	return decodeUnicodeEscapes(string(normalized)), nil
}

// decodeUnicodeEscapes replaces YAML/JSON Unicode escape sequences with their UTF-8 equivalents.
// This is needed because the K8s serialiser routes through gopkg.in/yaml.v2, which escapes
// non-BMP Unicode characters (codepoints > U+FFFF) as \UXXXXXXXX in double-quoted strings.
// Processing order matters: \UXXXXXXXX first, then surrogate pairs, then standalone \uXXXX.
func decodeUnicodeEscapes(s string) string {
	// Decode \UXXXXXXXX (8-digit YAML escapes for non-BMP codepoints)
	s = reUnicodeEscape8.ReplaceAllStringFunc(s, func(match string) string {
		codepoint, err := strconv.ParseUint(match[2:], 16, 32)
		if err != nil || !utf8.ValidRune(rune(codepoint)) {
			return match
		}
		return string(rune(codepoint))
	})

	// Decode \uXXXX\uXXXX (JSON-style UTF-16 surrogate pairs)
	s = reSurrogatePair.ReplaceAllStringFunc(s, func(match string) string {
		high, err1 := strconv.ParseUint(match[2:6], 16, 16)
		low, err2 := strconv.ParseUint(match[8:12], 16, 16)
		if err1 != nil || err2 != nil {
			return match
		}
		r := utf16.DecodeRune(rune(high), rune(low))
		if r == unicode.ReplacementChar {
			return match
		}
		return string(r)
	})

	// Decode standalone \uXXXX (BMP codepoints, skip surrogates)
	s = reUnicodeEscape4.ReplaceAllStringFunc(s, func(match string) string {
		codepoint, err := strconv.ParseUint(match[2:], 16, 16)
		if err != nil || (codepoint >= 0xD800 && codepoint <= 0xDFFF) {
			return match
		}
		if !utf8.ValidRune(rune(codepoint)) {
			return match
		}
		return string(rune(codepoint))
	})

	return s
}

// reNonBMP matches any Unicode character above U+FFFF (non-BMP codepoints).
var reNonBMP = regexp.MustCompile(`[\x{10000}-\x{10FFFF}]`)

// normalizeTemplateStyles ensures YAML strings containing template expressions
// (repl{{ or {{repl) are not double-quoted. The go.yaml.in/yaml/v3 library considers
// non-BMP Unicode characters (> U+FFFF) non-printable, forcing double-quoted style
// with \UXXXXXXXX escapes. This also introduces \" for any " in the value, which
// breaks Go's text/template parser when \" appears inside template delimiters.
//
// The approach: temporarily replace non-BMP characters with ASCII placeholders so
// the YAML encoder chooses single-quoted (or literal block) style, then restore
// the original UTF-8 characters in the encoded output.
func normalizeTemplateStyles(yamlBytes []byte) ([]byte, error) {
	var doc goyaml.Node
	if err := goyaml.Unmarshal(yamlBytes, &doc); err != nil {
		return nil, err
	}

	// Track placeholder→original mappings
	placeholders := map[string]string{}
	walkYAMLNodes(&doc, func(node *goyaml.Node) {
		if node.Kind != goyaml.ScalarNode {
			return
		}
		if !strings.Contains(node.Value, "repl{{") && !strings.Contains(node.Value, "{{repl") {
			return
		}
		// Replace non-BMP characters with ASCII placeholders
		node.Value = reNonBMP.ReplaceAllStringFunc(node.Value, func(ch string) string {
			r := []rune(ch)[0]
			placeholder := fmt.Sprintf("__KOTS_U%08X__", r)
			placeholders[placeholder] = ch
			return placeholder
		})
		// Now the value is BMP-only, so the encoder can use non-double-quoted styles
		if strings.Contains(node.Value, "\n") {
			node.Style = goyaml.LiteralStyle
		} else {
			node.Style = goyaml.SingleQuotedStyle
		}
	})

	var buf bytes.Buffer
	enc := goyaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, err
	}

	// Restore original UTF-8 characters
	result := buf.String()
	for placeholder, original := range placeholders {
		result = strings.ReplaceAll(result, placeholder, original)
	}

	return []byte(result), nil
}

func walkYAMLNodes(node *goyaml.Node, fn func(*goyaml.Node)) {
	if node == nil {
		return
	}
	fn(node)
	for _, child := range node.Content {
		walkYAMLNodes(child, fn)
	}
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
		shortUUID := strings.Split(uuid.New().String(), "-")[0]
		variadicName := fmt.Sprintf("%s-%s", item.Name, shortUUID)
		item.ValuesByGroup[groupName][variadicName] = ""
	}
}
