package validators

import (
	"context"
	_ "embed"

	"github.com/mitchellh/mapstructure"
	"github.com/open-policy-agent/opa/rego"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/lint/types"
	"github.com/replicatedhq/kots/pkg/lint/util"
)

//go:embed rego/kots-spec-opa-nonrendered.rego
var nonRenderedRegoContent string

var (
	// a prepared rego query for linting NON-rendered files
	nonRenderedRegoQuery *rego.PreparedEvalQuery
)

// InitOPA initializes the OPA linting queries
func InitOPA() error {
	ctx := context.Background()

	// prepare rego query for linting non-rendered files
	nonRenderedQuery, err := rego.New(
		rego.Query("data.kots.spec.nonrendered.lint"),
		rego.Module("kots-spec-opa-nonrendered.rego", string(nonRenderedRegoContent)),
	).PrepareForEval(ctx)

	if err != nil {
		return errors.Wrap(err, "failed to prepare non-rendered query for eval")
	}

	nonRenderedRegoQuery = &nonRenderedQuery

	return nil
}

// ValidateOPANonRendered validates files using OPA policies before rendering
// InitOPA needs to be called first in order for this function to run successfully
func ValidateOPANonRendered(specFiles types.SpecFiles) ([]types.LintExpression, error) {
	separatedSpecFiles, err := specFiles.Separate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to separate multi docs")
	}

	ctx := context.Background()
	results, err := nonRenderedRegoQuery.Eval(ctx, rego.EvalInput(separatedSpecFiles))
	if err != nil {
		return nil, errors.Wrap(err, "failed to evaluate query")
	}

	return opaResultsToLintExpressions(results, specFiles)
}

func opaResultsToLintExpressions(results rego.ResultSet, specFiles types.SpecFiles) ([]types.LintExpression, error) {
	lintExpressions := []types.LintExpression{}

	if len(results) == 0 {
		return lintExpressions, nil
	}

	result := results[0]
	if len(result.Expressions) == 0 {
		return lintExpressions, nil
	}

	var opaLintExpressions []types.OPALintExpression
	if err := mapstructure.Decode(result.Expressions[0].Value, &opaLintExpressions); err != nil {
		return nil, errors.Wrap(err, "failed to mapstructure opa lint expressions")
	}

	// map opa lint expressions to standard lint expressions
	for _, opaLintExpression := range opaLintExpressions {
		lintExpression := types.LintExpression{
			Rule:    opaLintExpression.Rule,
			Type:    opaLintExpression.Type,
			Message: opaLintExpression.Message,
		}

		if opaLintExpression.Path == "" {
			lintExpressions = append(lintExpressions, lintExpression)
			continue
		}

		lintExpression.Path = opaLintExpression.Path

		// we need to get the line number for the original file content not the separated document nor the rendered one
		foundSpecFile, err := specFiles.GetFile(opaLintExpression.Path)
		if err != nil {
			lintExpressions = append(lintExpressions, lintExpression)
			continue
		}

		line := -1
		if opaLintExpression.Field != "" {
			line, _ = util.GetLineNumberFromYamlPath(foundSpecFile.Content, opaLintExpression.Field, opaLintExpression.DocIndex)
		} else if opaLintExpression.Match != "" {
			line, _ = util.GetLineNumberFromMatch(foundSpecFile.Content, opaLintExpression.Match, opaLintExpression.DocIndex)
		} else if opaLintExpression.Type == "error" {
			line, _ = util.GetLineNumberForDoc(foundSpecFile.Content, opaLintExpression.DocIndex)
		}

		if line == -1 {
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

	return lintExpressions, nil
}
