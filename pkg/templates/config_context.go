package templates

import (
	"encoding/base64"
	"fmt"
	"text/template"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
)

func (b *Builder) NewConfigContext(configGroups []kotsv1beta1.ConfigGroup, templateContext map[string]interface{}) (*ConfigCtx, error) {
	configCtx := &ConfigCtx{
		ItemValues: templateContext,
	}

	for _, configGroup := range configGroups {
		for _, configItem := range configGroup.Items {
			// if the pending value is different from the built, then use the pending every time
			// We have to ignore errors here because we only have the static context loaded
			// for rendering. some items have templates that need the config context,
			// so we can ignore these.
			builtDefault, _ := b.String(configItem.Default)
			builtValue, _ := b.String(configItem.Value)

			var built string
			if builtValue != "" {
				built = builtValue
			} else {
				built = builtDefault
			}

			if v, ok := templateContext[configItem.Name]; ok {
				built = fmt.Sprintf("%v", v)
			}

			configCtx.ItemValues[configItem.Name] = built
		}
	}

	return configCtx, nil
}

// ConfigCtx is the context for builder functions before the application has started.
type ConfigCtx struct {
	ItemValues map[string]interface{}
}

// FuncMap represents the available functions in the ConfigCtx.
func (ctx ConfigCtx) FuncMap() template.FuncMap {
	return template.FuncMap{
		"ConfigOption":          ctx.configOption,
		"ConfigOptionIndex":     ctx.configOptionIndex,
		"ConfigOptionData":      ctx.configOptionData,
		"ConfigOptionEquals":    ctx.configOptionEquals,
		"ConfigOptionNotEquals": ctx.configOptionNotEquals,
	}
}

func (ctx ConfigCtx) configOption(name string) string {
	v, err := ctx.getConfigOptionValue(name)
	if err != nil {
		return ""
	}
	return v
}

func (ctx ConfigCtx) configOptionIndex(name string) string {
	return ""
}

func (ctx ConfigCtx) configOptionData(name string) string {
	v, err := ctx.getConfigOptionValue(name)
	if err != nil {
		return ""
	}

	decoded, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return ""
	}

	return string(decoded)
}

func (ctx ConfigCtx) configOptionEquals(name string, value string) bool {
	val, err := ctx.getConfigOptionValue(name)
	if err != nil {
		return false
	}

	return value == val
}

func (ctx ConfigCtx) configOptionNotEquals(name string, value string) bool {
	val, err := ctx.getConfigOptionValue(name)
	if err != nil {
		return false
	}

	return value != val
}

func (ctx ConfigCtx) getConfigOptionValue(itemName string) (string, error) {
	if val, ok := ctx.ItemValues[itemName]; ok {
		return fmt.Sprintf("%v", val), nil
	}

	return "", errors.New("unable to find config item")
}
