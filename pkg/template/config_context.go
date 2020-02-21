package template

import (
	"encoding/base64"
	"fmt"
	"text/template"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
)

func (b *Builder) NewConfigContext(configGroups []kotsv1beta1.ConfigGroup, existingValues map[string]ItemValue, cipher *crypto.AESCipher) (*ConfigCtx, error) {
	configCtx := &ConfigCtx{
		ItemValues: existingValues,
	}

	builder := Builder{
		Ctx: []Ctx{
			configCtx,
			StaticCtx{},
		},
	}

	configItemsByName := make(map[string]kotsv1beta1.ConfigItem)
	for _, configGroup := range configGroups {
		for _, configItem := range configGroup.Items {
			configItemsByName[configItem.Name] = configItem
		}
	}

	deps := depGraph{}
	err := deps.ParseConfigGroup(configGroups) // this updates the 'deps' object to include a dependency graph
	if err != nil {
		return nil, errors.Wrap(err, "generate config groups dep graph")
	}

	var headNodes []string
	headNodes, err = deps.GetHeadNodes() // get the list of config items that do not depend on unresolved config items
	for (len(headNodes) > 0) && (err == nil) {
		for _, node := range headNodes {
			deps.ResolveDep(node)

			configItem := configItemsByName[node]

			if !isReadOnly(configItem) {
				// if item is editable and the live state is valid, skip the rest of this -
				val, ok := configCtx.ItemValues[node]
				if ok && val.HasValue() {
					continue
				}
			}

			// build "default" and "value"
			builtDefault, _ := builder.String(configItem.Default.String())
			builtValue, _ := builder.String(configItem.Value.String())
			itemValue := ItemValue{
				Value:   builtValue,
				Default: builtDefault,
			}

			//
			if configItem.Type == "password" && itemValue.HasValue() {
				// FIXME: this temporarily ignores errors and falls back on old behavior
				val, err := decrypt(itemValue.ValueStr(), cipher)
				if err == nil {
					itemValue.Value = val
				}
			}
			configCtx.ItemValues[configItem.Name] = itemValue
		}

		// update headNodes list for next loop iteration
		headNodes, err = deps.GetHeadNodes()
	}
	if err != nil {
		// dependencies could not be resolved for some reason
		// return the empty config
		// TODO: Better error messaging
		return &ConfigCtx{}, err
	}
	return configCtx, nil
}

// isReadOnly checks to see if it should be possible to edit a field
// for instance, it should not be possible to edit the value of a label
func isReadOnly(item kotsv1beta1.ConfigItem) bool {
	if item.ReadOnly {
		return true
	}

	// "" is an editable type because the default type is "text"
	var EditableItemTypes = map[string]struct{}{
		"":            {},
		"bool":        {},
		"file":        {},
		"password":    {},
		"select":      {},
		"select_many": {},
		"select_one":  {},
		"text":        {},
		"textarea":    {},
	}

	_, editable := EditableItemTypes[item.Type]
	return !editable
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
		"LocalRegistryAddress":  ctx.localRegistryAddress,
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

func (ctx ConfigCtx) localRegistryAddress() string {
	return ""
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
