package template

import (
	"encoding/base64"
	"fmt"
	"path"
	"regexp"
	"text/template"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/image"
	corev1 "k8s.io/api/core/v1"
)

var (
	dockerImageNameRegex = regexp.MustCompile("(?:([^\\/]+)\\/)?(?:([^\\/]+)\\/)?([^@:\\/]+)(?:[@:](.+))")
)

type LocalRegistry struct {
	Host      string
	Namespace string
	Username  string
	Password  string
}

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

// ConfigCtx is the context for builder functions before the application has started.
type ConfigCtx struct {
	ItemValues    map[string]ItemValue
	LocalRegistry LocalRegistry

	license         *kotsv1beta1.License // Another agument for unifying all these contexts
	unsignedLicense *kotsv1beta1.UnsignedLicense
}

// newConfigContext creates and returns a context for template rendering
func (b *Builder) newConfigContext(configGroups []kotsv1beta1.ConfigGroup, existingValues map[string]ItemValue, localRegistry LocalRegistry, cipher *crypto.AESCipher, license *kotsv1beta1.License, unsignedLicense *kotsv1beta1.UnsignedLicense) (*ConfigCtx, error) {
	configCtx := &ConfigCtx{
		ItemValues:      existingValues,
		LocalRegistry:   localRegistry,
		license:         license,
		unsignedLicense: unsignedLicense,
	}

	builder := Builder{
		Ctx: []Ctx{
			configCtx,
			StaticCtx{},
			&LicenseCtx{License: license},
			NewKurlContext("base", "default"),
		},
	}

	configItemsByName := make(map[string]kotsv1beta1.ConfigItem)
	for _, configGroup := range configGroups {
		for _, configItem := range configGroup.Items {
			configItemsByName[configItem.Name] = configItem

			// decrypt password if it exists
			if configItem.Type == "password" {
				existingVal, ok := existingValues[configItem.Name]
				if ok && existingVal.HasValue() {
					val, err := decrypt(existingVal.ValueStr(), cipher)
					if err == nil {
						existingVal.Value = val
						existingValues[configItem.Name] = existingVal
					}
				}
			}
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
				_, ok := configCtx.ItemValues[node]
				if ok {
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

// FuncMap represents the available functions in the ConfigCtx.
func (ctx ConfigCtx) FuncMap() template.FuncMap {
	return template.FuncMap{
		"ConfigOption":                 ctx.configOption,
		"ConfigOptionIndex":            ctx.configOptionIndex,
		"ConfigOptionData":             ctx.configOptionData,
		"ConfigOptionEquals":           ctx.configOptionEquals,
		"ConfigOptionNotEquals":        ctx.configOptionNotEquals,
		"LocalRegistryAddress":         ctx.localRegistryAddress,
		"LocalRegistryHost":            ctx.localRegistryHost,
		"LocalRegistryNamespace":       ctx.localRegistryNamespace,
		"LocalImageName":               ctx.localImageName,
		"LocalRegistryImagePullSecret": ctx.localRegistryImagePullSecret,
		"HasLocalRegistry":             ctx.hasLocalRegistry,
	}
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
	if ctx.LocalRegistry.Namespace == "" {
		return ctx.LocalRegistry.Host
	}

	return fmt.Sprintf("%s/%s", ctx.LocalRegistry.Host, ctx.LocalRegistry.Namespace)
}

func (ctx ConfigCtx) localRegistryHost() string {
	return ctx.LocalRegistry.Host
}

func (ctx ConfigCtx) localRegistryNamespace() string {
	return ctx.LocalRegistry.Namespace
}

func (ctx ConfigCtx) localImageName(imageRef string) string {
	// If there's a private registry. Always rewrite everything.  This covers airgap installs too.
	if ctx.LocalRegistry.Host != "" {
		ref, err := image.RefFromImage(imageRef)
		if err != nil {
			// TODO: log
			return ""
		}
		ref.Domain = ctx.localRegistryHost()
		ref.Name = path.Join(ctx.localRegistryAddress(), ref.NameBase())
		return ref.String()
	}

	// Not airgap and no local registry.  Rewrite images that are private only.

	isPrivate, err := image.IsPrivateImage(imageRef)
	if err != nil {
		// TODO: log
		return ""
	}

	if !isPrivate {
		return imageRef
	}

	proxyInfo := registry.ProxyEndpointFromLicense(ctx.license)
	registryOptions := registry.RegistryOptions{
		Endpoint:      proxyInfo.Registry,
		ProxyEndpoint: proxyInfo.Proxy,
	}

	licenseAppSlug := ""
	if ctx.license != nil {
		licenseAppSlug = ctx.license.Spec.AppSlug
	}

	newImage, err := image.RewritePrivateImage(registryOptions, imageRef, licenseAppSlug)
	if err != nil {
		// TODO: log
		return ""
	}

	return newImage
}

func (ctx ConfigCtx) hasLocalRegistry() bool {
	return ctx.LocalRegistry.Host != ""
}

func (ctx ConfigCtx) localRegistryImagePullSecret() string {
	var secret *corev1.Secret
	if ctx.LocalRegistry.Host != "" {
		s, err := registry.PullSecretForRegistries(
			[]string{ctx.LocalRegistry.Host},
			ctx.LocalRegistry.Username,
			ctx.LocalRegistry.Password,
			"default", // this value doesn't matter
		)
		if err != nil {
			return ""
		}
		secret = s
	} else {
		licenseIDString := ""
		if ctx.license != nil {
			licenseIDString = ctx.license.Spec.LicenseID
		}

		proxyInfo := registry.ProxyEndpointFromLicense(ctx.license)
		s, err := registry.PullSecretForRegistries(
			proxyInfo.ToSlice(),
			licenseIDString,
			licenseIDString,
			"default", // this value doesn't matter
		)
		if err != nil {
			return ""
		}
		secret = s
	}
	dockerConfig, found := secret.Data[".dockerconfigjson"]
	if !found {
		return ""
	}

	return base64.StdEncoding.EncodeToString(dockerConfig)
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
