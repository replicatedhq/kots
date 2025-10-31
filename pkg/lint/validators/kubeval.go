package validators

import (
	"fmt"
	"strings"

	"github.com/instrumenta/kubeval/kubeval"
	"github.com/replicatedhq/kots/pkg/lint/schema"
	"github.com/replicatedhq/kots/pkg/lint/types"
	"github.com/replicatedhq/kots/pkg/lint/util"
)

// ValidateKubernetes validates Kubernetes resources using kubeval
// renderedFiles are the rendered files to be linted (we don't render on the fly because it is an expensive process)
// originalFiles are the non-rendered non-separated files, which are needed to find the actual line number
func ValidateKubernetes(renderedFiles, originalFiles types.SpecFiles) ([]types.LintExpression, error) {
	schemaLocation := fmt.Sprintf("file://%s", schema.GetSchemaDir())
	return validateWithKubeval(renderedFiles, originalFiles, schemaLocation)
}

// validateWithKubeval validates files using kubeval with a specific schema location
func validateWithKubeval(renderedFiles, originalFiles types.SpecFiles, schemaLocation string) ([]types.LintExpression, error) {
	lintExpressions := []types.LintExpression{}

	kubevalConfig := kubeval.Config{
		SchemaLocation:       schemaLocation,
		Strict:               true,
		KubernetesVersion:    schema.KubernetesLintVersion,
		IgnoreMissingSchemas: true,
		DefaultNamespace:     "default",
	}

	for _, renderedFile := range renderedFiles {
		if !renderedFile.IsYAML() {
			continue
		}

		kubevalConfig.FileName = renderedFile.Path
		results, err := kubeval.Validate([]byte(renderedFile.Content), &kubevalConfig)
		if err != nil {
			var lintExpression types.LintExpression

			if strings.Contains(err.Error(), "Failed initalizing schema") && strings.Contains(err.Error(), "no such file or directory") {
				lintExpression = types.LintExpression{
					Rule:    "kubeval-schema-not-found",
					Type:    "warn",
					Path:    renderedFile.Path,
					Message: "We currently have no matching schema to lint this type of file",
				}
			} else {
				lintExpression = types.LintExpression{
					Rule:    "kubeval-error",
					Type:    "error",
					Path:    renderedFile.Path,
					Message: err.Error(),
				}
			}

			lintExpressions = append(lintExpressions, lintExpression)

			continue // don't stop
		}

		for _, validationResult := range results {
			for _, validationError := range validationResult.Errors {
				lintExpression := types.LintExpression{
					Rule:    validationError.Type(),
					Type:    "warn",
					Path:    renderedFile.Path,
					Message: validationError.Description(),
				}

				// we need to get the line number for the original file content
				// not the rendered version of it, and not the separated document
				yamlPath := validationError.Field()
				foundSpecFile, err := originalFiles.GetFile(renderedFile.Path)
				if err != nil {
					lintExpressions = append(lintExpressions, lintExpression)
					continue
				}

				line, err := util.GetLineNumberFromYamlPath(foundSpecFile.Content, yamlPath, renderedFile.DocIndex)
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

				lintExpressions = append(lintExpressions, lintExpression)
			}
		}
	}

	return lintExpressions, nil
}
