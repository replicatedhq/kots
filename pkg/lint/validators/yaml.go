package validators

import (
	"bytes"
	"io"

	"github.com/replicatedhq/kots/pkg/lint/types"
	"github.com/replicatedhq/kots/pkg/lint/util"
	"gopkg.in/yaml.v2"
)

// ValidateYAML validates that all YAML files have valid syntax
func ValidateYAML(files types.SpecFiles) []types.LintExpression {
	lintExpressions := []types.LintExpression{}

	// all files must be valid YAML, so without a schema, attempt to parse them
	// we do this separately because it's really hard to get kubeval to
	// return valid errors on all types of invalid yaml

	for _, specFile := range files {
		fileLintExpressions := lintFileHasValidYAML(specFile)
		lintExpressions = append(lintExpressions, fileLintExpressions...)
	}

	return lintExpressions
}

func lintFileHasValidYAML(file types.SpecFile) []types.LintExpression {
	lintExpressions := []types.LintExpression{}

	if !file.IsYAML() {
		return lintExpressions
	}

	reader := bytes.NewReader([]byte(file.Content))
	decoder := yaml.NewDecoder(reader)
	decoder.SetStrict(true)

	for {
		var doc interface{}
		err := decoder.Decode(&doc)

		if err == nil {
			continue
		}

		if err == io.EOF {
			break
		}

		lintExpression := types.LintExpression{
			Rule:    "invalid-yaml",
			Type:    "error",
			Path:    file.Path,
			Message: err.Error(),
		}

		line, err := util.TryGetLineNumberFromValue(err.Error())
		if err == nil && line > -1 {
			lintExpression.Positions = []types.LintExpressionItemPosition{
				{
					Start: types.LintExpressionItemLinePosition{
						Line: line,
					},
				},
			}
		}

		lintExpressions = append(lintExpressions, lintExpression)

		break // break on first error, otherwise decoder will panic
	}

	return lintExpressions
}

// ValidateRenderedYAML validates that rendered files are still valid YAML
func ValidateRenderedYAML(renderedFiles types.SpecFiles) []types.LintExpression {
	var lintExpressions []types.LintExpression

	for _, renderedFile := range renderedFiles {
		if !renderedFile.IsYAML() {
			continue
		}

		var doc interface{}
		err := yaml.Unmarshal([]byte(renderedFile.Content), &doc)
		if err != nil {
			lintErrMsg := err.Error()
			errLine, lineErr := util.TryGetLineNumberFromValue(err.Error())
			if lineErr == nil && errLine > -1 {
				lines := bytes.Split([]byte(renderedFile.Content), []byte("\n"))
				if len(lines) > errLine {
					errLineContent := string(bytes.TrimSpace(lines[errLine-1]))
					// Remove the "line N:" part from the error message
					lintErrMsg = err.Error() + ": " + errLineContent
				}
			}

			lintExpression := types.LintExpression{
				Rule:    "invalid-rendered-yaml",
				Type:    "error",
				Path:    renderedFile.Path,
				Message: lintErrMsg,
			}
			lintExpressions = append(lintExpressions, lintExpression)
		}
	}

	return lintExpressions
}
