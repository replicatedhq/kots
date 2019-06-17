package templates

import (
	"encoding/base64"
	"fmt"
	"text/template"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/libyaml"
)

func (bb *BuilderBuilder) NewConfigContext(configGroups []libyaml.ConfigGroup, templateContext map[string]interface{}) (*ConfigCtx, error) {
	builder := bb.NewBuilder(
		bb.NewStaticContext(),
	)

	configCtx := &ConfigCtx{
		ItemValues: templateContext,
		Logger:     bb.Logger,
	}

	for _, configGroup := range configGroups {
		for _, configItem := range configGroup.Items {
			// if the pending value is different from the built, then use the pending every time
			// We have to ignore errors here because we only have the static context loaded
			// for rendering. some items have templates that need the config context,
			// so we can ignore these.
			builtDefault, _ := builder.String(configItem.Default)
			builtValue, _ := builder.String(configItem.Value)

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
	Logger     log.Logger
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

	err := fmt.Errorf("unable to find config item named %q", itemName)
	level.Error(ctx.Logger).Log("msg", "unable to find config item", "err", err)
	return "", err
}
