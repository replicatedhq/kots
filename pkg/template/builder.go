package template

import (
	"bytes"
	"regexp"
	"strconv"
	"text/template"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	dockerregistrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
)

var (
	templateNotDefinedRegexp = regexp.MustCompile(`template.*not defined$`)
)

type Builder struct {
	Ctx    []Ctx
	Functs template.FuncMap
}

type BuilderOptions struct {
	ConfigGroups    []kotsv1beta1.ConfigGroup
	ExistingValues  map[string]ItemValue
	LocalRegistry   registrytypes.RegistrySettings
	License         *kotsv1beta1.License
	Application     *kotsv1beta1.Application
	ApplicationInfo *ApplicationInfo
	VersionInfo     *VersionInfo
	IdentityConfig  *kotsv1beta1.IdentityConfig
	Namespace       string
	DecryptValues   bool
}

// NewBuilder creates a builder with all available contexts.
func NewBuilder(opts BuilderOptions) (Builder, map[string]ItemValue, error) {
	b := Builder{}

	// do not fail on being unable to get dockerhub credentials, since they're just used to increase the rate limit
	dockerHubRegistry := dockerregistrytypes.RegistryOptions{}
	if opts.Namespace != "" {
		clientset, err := k8sutil.GetClientset()
		if err == nil {
			dockerHubRegistryCreds, _ := registry.GetDockerHubCredentials(clientset, opts.Namespace)
			dockerHubRegistry = dockerregistrytypes.RegistryOptions{
				Username: dockerHubRegistryCreds.Username,
				Password: dockerHubRegistryCreds.Password,
			}
		}
	}

	slug := ""
	if opts.ApplicationInfo != nil {
		slug = opts.ApplicationInfo.Slug
	}

	configCtx, err := b.newConfigContext(opts.ConfigGroups, opts.ExistingValues, opts.LocalRegistry,
		opts.License, opts.Application, opts.VersionInfo, dockerHubRegistry, slug, opts.DecryptValues)
	if err != nil {
		return Builder{}, nil, errors.Wrap(err, "create config context")
	}

	b.Ctx = []Ctx{
		StaticCtx{},
		licenseCtx{License: opts.License, App: opts.Application, VersionInfo: opts.VersionInfo},
		newKurlContext("base", "default"), // can be hardcoded because kurl always deploys to the default namespace
		newVersionCtx(opts.VersionInfo),
		newIdentityCtx(opts.IdentityConfig, opts.ApplicationInfo),
		configCtx,
	}
	return b, configCtx.ItemValues, nil
}

func (b *Builder) AddCtx(ctx Ctx) {
	b.Ctx = append(b.Ctx, ctx)
}

func (b *Builder) String(text string) (string, error) {
	if text == "" {
		return "", nil
	}
	result, err := b.RenderTemplate(text, text)
	return result, err
}

func (b *Builder) Bool(text string, defaultVal bool) (bool, error) {
	if text == "" {
		return defaultVal, nil
	}

	value, err := b.RenderTemplate(text, text)
	if err != nil {
		return defaultVal, errors.Wrap(err, "failed to render bool template")
	}

	// If the template didn't parse (turns into an empty string), then we should
	// return the default
	if value == "" {
		return defaultVal, nil
	}

	result, err := strconv.ParseBool(value)
	if err != nil {
		// for now we are assuming default value if we fail to parse
		return defaultVal, nil
	}

	return result, nil
}

func (b *Builder) Int(text string, defaultVal int64) (int64, error) {
	if text == "" {
		return defaultVal, nil
	}

	value, err := b.RenderTemplate(text, text)
	if err != nil {
		return defaultVal, errors.Wrap(err, "failed to render int template")
	}

	// If the template didn't parse (turns into an empty string), then we should
	// return the default
	if value == "" {
		return defaultVal, nil
	}

	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		// for now we are assuming default value if we fail to parse
		return defaultVal, nil
	}

	return result, nil
}

func (b *Builder) Uint(text string, defaultVal uint64) (uint64, error) {
	if text == "" {
		return defaultVal, nil
	}

	value, err := b.RenderTemplate(text, text)
	if err != nil {
		return defaultVal, errors.Wrap(err, "failed to render uint template")
	}

	// If the template didn't parse (turns into an empty string), then we should
	// return the default
	if value == "" {
		return defaultVal, nil
	}

	result, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		// for now we are assuming default value if we fail to parse
		return defaultVal, nil
	}

	return result, nil
}

func (b *Builder) Float64(text string, defaultVal float64) (float64, error) {
	if text == "" {
		return defaultVal, nil
	}

	value, err := b.RenderTemplate(text, text)
	if err != nil {
		return defaultVal, errors.Wrap(err, "failed to render float template")
	}

	// If the template didn't parse (turns into an empty string), then we should
	// return the default
	if value == "" {
		return defaultVal, nil
	}

	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		// for now we are assuming default value if we fail to parse
		return defaultVal, nil
	}

	return result, nil
}

func (b *Builder) BuildFuncMap() template.FuncMap {
	if b.Functs == nil {
		b.Functs = template.FuncMap{}
	}

	funcMap := b.Functs
	for _, ctx := range b.Ctx {
		for name, fn := range ctx.FuncMap() {
			funcMap[name] = fn
		}
	}
	return funcMap
}

func (b *Builder) GetTemplate(name, text string, rdelim, ldelim string) (*template.Template, error) {
	tmpl, err := template.New(name).Delims(rdelim, ldelim).Funcs(b.BuildFuncMap()).Parse(text)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

func (b *Builder) RenderTemplate(name string, text string) (string, error) {
	delims := []struct {
		rdelim string
		ldelim string
	}{
		{"{{repl", "}}"},
		{"repl{{", "}}"},
	}

	curText := text
	for _, d := range delims {
		tmpl, err := b.GetTemplate(name, curText, d.rdelim, d.ldelim)
		if err != nil {
			return "", errors.Wrap(err, "failed to get template")
		}

		var contents bytes.Buffer
		if err := tmpl.Execute(&contents, nil); err != nil {
			return "", errors.Wrap(err, "failed to execute template")
		}
		curText = contents.String()
	}

	return curText, nil
}
