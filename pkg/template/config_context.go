package template

import (
	"encoding/base64"
	"fmt"
	"text/template"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
)

func (b *Builder) NewConfigContext(configGroups []kotsv1beta1.ConfigGroup, templateContext map[string]ItemValue, cipher *crypto.AESCipher) (*ConfigCtx, error) {
	configCtx := &ConfigCtx{
		ItemValues: templateContext,
	}

	for _, configGroup := range configGroups {
		for _, configItem := range configGroup.Items {
			// if the pending value is different from the built, then use the pending every time
			// We have to ignore errors here because we only have the static context loaded
			// for rendering. some items have templates that need the config context,
			// so we can ignore these.

			var itemValue ItemValue
			if v, ok := templateContext[configItem.Name]; ok {
				itemValue = ItemValue{
					Value:   v.Value,
					Default: v.Default,
				}
			} else {
				builtDefault, _ := b.String(configItem.Default.String())
				builtValue, _ := b.String(configItem.Value.String())
				itemValue = ItemValue{
					Value:   builtValue,
					Default: builtDefault,
				}
			}

			if configItem.Type == "password" && itemValue.HasValue() {
				// FIXME: this temporarily ignores errors and falls back on old behavior
				val, err := decrypt(itemValue.ValueStr(), cipher)
				if err == nil {
					itemValue.Value = val
				}
			}
			configCtx.ItemValues[configItem.Name] = itemValue
		}
	}

	return configCtx, nil
}

// ConfigCtx is the context for builder functions before the application has started.
type ItemValue struct {
	Value   interface{}
	Default interface{}
}

func (i ItemValue) HasValue() bool {
	if v, ok := i.Value.(string); ok {
		return v != ""
	}
	return i.Value != nil
}

func (i ItemValue) ValueStr() string {
	if i.HasValue() {
		return fmt.Sprintf("%s", i.Value)
	}
	return ""
}

func (i ItemValue) HasDefault() bool {
	if v, ok := i.Default.(string); ok {
		return v != ""
	}
	return i.Default != nil
}

func (i ItemValue) DefaultStr() string {
	if i.HasDefault() {
		return fmt.Sprintf("%s", i.Default)
	}
	return ""
}

type ConfigCtx struct {
	ItemValues map[string]ItemValue
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
	val, ok := ctx.ItemValues[itemName]
	if !ok {
		return "", errors.New("unable to find config item")
	}

	if val.HasValue() {
		return val.ValueStr(), nil
	}

	return val.DefaultStr(), nil
}

func decrypt(input string, cipher *crypto.AESCipher) (string, error) {
	if cipher == nil {
		return "", errors.New("cipher not defined")
	}

	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", errors.Wrap(err, "failed to base64 decode")
	}

	decrypted, err := cipher.Decrypt(decoded)
	if err != nil {
		return "", errors.Wrap(err, "failed to decrypt")
	}

	return string(decrypted), nil
}
