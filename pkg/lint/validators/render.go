package validators

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/lint/types"
	"github.com/replicatedhq/kots/pkg/lint/util"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/template"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	"gopkg.in/yaml.v2"
	"k8s.io/kubectl/pkg/scheme"
)

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
}

// RenderTemplateError represents an error that occurred during template rendering
type RenderTemplateError struct {
	message string
	match   string
}

func (r RenderTemplateError) Error() string {
	return r.message
}

func (r RenderTemplateError) Match() string {
	return r.match
}

// ValidateRendering validates that templates can be rendered successfully
func ValidateRendering(specFiles types.SpecFiles) ([]types.LintExpression, types.SpecFiles, error) {
	lintExpressions := []types.LintExpression{}

	separatedSpecFiles, err := specFiles.Separate()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to separate multi docs")
	}

	// check if config is valid
	kotsConfig, path, err := findAndValidateConfig(separatedSpecFiles)
	if err != nil {
		lintExpression := types.LintExpression{
			Rule:    "config-is-invalid",
			Type:    "error",
			Path:    path,
			Message: err.Error(),
		}
		lintExpressions = append(lintExpressions, lintExpression)
	}

	builder, err := getTemplateBuilder(kotsConfig)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get template builder")
	}

	// rendering files is an expensive process, store and return the rendered files
	// from this function so that they can be used later instead of rendering again on the fly
	renderedFiles := types.SpecFiles{}

	for _, file := range separatedSpecFiles {
		renderedContent, err := renderContent(file, builder)
		if err == nil {
			file.Content = string(renderedContent)
			renderedFiles = append(renderedFiles, file)
			continue
		}
		// check if the error is coming from kots RenderTemplate function
		if renderErr, ok := errors.Cause(err).(RenderTemplateError); ok {
			lintExpression := types.LintExpression{
				Rule:    "unable-to-render",
				Type:    "error",
				Path:    file.Path,
				Message: renderErr.Error(),
			}

			if renderErr.Match() != "" {
				// we need to get the line number for the original file content not the separated document
				foundSpecFile, err := specFiles.GetFile(file.Path)
				if err != nil {
					lintExpressions = append(lintExpressions, lintExpression)
					continue
				}

				line, err := util.GetLineNumberFromMatch(foundSpecFile.Content, renderErr.Match(), file.DocIndex)
				if err != nil || line == -1 {
					lintExpressions = append(lintExpressions, lintExpression)
					continue
				}
				lintExpression.Positions = []types.LintExpressionItemPosition{
					{
						Start: types.LintExpressionItemLinePosition{
							Line: line,
						},
					},
				}
			}

			lintExpressions = append(lintExpressions, lintExpression)
			continue
		}
		// error is not caused by kots RenderTemplate, something went wrong
		return nil, nil, errors.Wrapf(err, "failed to render spec file %s", file.Path)
	}

	return lintExpressions, renderedFiles, nil
}

func findAndValidateConfig(files types.SpecFiles) (*kotsv1beta1.Config, string, error) {
	var kotsConfig *kotsv1beta1.Config
	var path string

	for _, file := range files {
		document := &types.GVKDoc{}
		if err := yaml.Unmarshal([]byte(file.Content), document); err != nil {
			continue
		}

		if document.APIVersion != "kots.io/v1beta1" || document.Kind != "Config" {
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode([]byte(file.Content), nil, nil)
		if err != nil {
			return nil, file.Path, errors.Wrap(err, "failed to decode config content")
		}

		if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Config" {
			kotsConfig = obj.(*kotsv1beta1.Config)
			path = file.Path
		}
	}

	if kotsConfig != nil {
		// if config was found, validate that it renders successfully
		configCopy := kotsConfig.DeepCopy()
		if _, err := renderConfig(configCopy); err != nil {
			return kotsConfig, path, errors.Wrap(err, "failed to render config")
		}
	}

	return kotsConfig, path, nil
}

func renderConfig(kotsConfig *kotsv1beta1.Config) ([]byte, error) {
	localRegistry := registrytypes.RegistrySettings{}
	appInfo := template.ApplicationInfo{}
	configValues := map[string]template.ItemValue{}

	renderedConfig, err := config.TemplateConfigObjects(kotsConfig, configValues, nil, nil, localRegistry, nil, &appInfo, nil, "", false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to template config objects")
	}

	b, err := yaml.Marshal(renderedConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal rendered config")
	}

	return b, nil
}

func getTemplateBuilder(kotsConfig *kotsv1beta1.Config) (*template.Builder, error) {
	localRegistry := registrytypes.RegistrySettings{}
	templateContextValues := make(map[string]template.ItemValue)

	configGroups := []kotsv1beta1.ConfigGroup{}
	if kotsConfig != nil && kotsConfig.Spec.Groups != nil {
		configGroups = kotsConfig.Spec.Groups
	}

	opts := template.BuilderOptions{
		ConfigGroups:   configGroups,
		ExistingValues: templateContextValues,
		LocalRegistry:  localRegistry,
		ApplicationInfo: &template.ApplicationInfo{ // Kots 1.56.0 calls ApplicationInfo.Slug, this is required
			Slug: "app-slug",
		},
	}
	builder, _, err := template.NewBuilder(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create builder")
	}

	return &builder, nil
}

func renderContent(file types.SpecFile, builder *template.Builder) ([]byte, error) {
	if !file.IsYAML() {
		return nil, errors.New("not a yaml file")
	}

	shouldRender, err := shouldBeRendered(file)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if file should be rendered")
	}
	if !shouldRender {
		return []byte(file.Content), nil
	}

	// add new line so that parsing the render template error is easier (possible)
	content := file.Content + "\n"

	rendered, err := builder.RenderTemplate(content, content)
	if err != nil {
		return nil, parseRenderTemplateError(file, err.Error())
	}

	// remove the new line that was added to make parsing template error easier (possible)
	rendered = strings.TrimSuffix(rendered, "\n")

	return []byte(rendered), nil
}

func shouldBeRendered(file types.SpecFile) (bool, error) {
	document := &types.GVKDoc{}
	if err := yaml.Unmarshal([]byte(file.Content), document); err != nil {
		// If YAML is invalid, assume it should be rendered (will fail in rendering step)
		// This prevents crashes when YAML syntax validator already caught the error
		return true, nil
	}

	if document.APIVersion == "kots.io/v1beta1" && document.Kind == "Config" {
		return false, nil
	}

	return true, nil
}

func parseRenderTemplateError(file types.SpecFile, value string) RenderTemplateError {
	/*
		** SAMPLE **
		failed to get template: template: apiVersion: v1
		data:
			ENV_VAR_1: fake
			ENV_VAR_2: '{{repl ConfigOptionEquals "test}}'
		kind: ConfigMap
		metadata:
			name: example-config
		:4: unterminated quoted string
	*/

	renderTemplateError := RenderTemplateError{
		match:   "",
		message: value,
	}

	parts := strings.Split(value, "\n:")
	if len(parts) == 1 {
		return renderTemplateError
	}

	lineAndMsg := parts[len(parts)-1]
	lineAndMsgParts := strings.SplitN(lineAndMsg, ":", 2)

	if len(lineAndMsgParts) == 1 {
		return renderTemplateError
	}

	// in some cases, the message contains the whole file content which is noisy and difficult to read
	msg := lineAndMsgParts[1]
	if i := strings.Index(msg, `\n"`); i != -1 {
		msg = msg[i+len(`\n"`):]
	}
	renderTemplateError.message = strings.TrimSpace(msg)

	// get the line number in the remarshalled (keys rearranged) data
	lineNumber, err := strconv.Atoi(lineAndMsgParts[0])
	if err != nil {
		return renderTemplateError
	}

	// try to find the data after it's been remarshalled (keys rearranged)
	data := util.GetStringInBetween(value, ": template: ", "\n:")
	if data == "" {
		return renderTemplateError
	}

	// find error line from data
	match := ""
	for index, line := range strings.Split(data, "\n") {
		if index == lineNumber-1 {
			match = line
			break
		}
	}
	renderTemplateError.match = match

	return renderTemplateError
}
