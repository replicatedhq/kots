package print

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/lint/types"
	"gopkg.in/yaml.v2"
)

// LintResult prints the lint result in the specified format
func LintResult(result *types.LintResult, format string, verbose bool) error {
	// Filter out info messages unless verbose
	filteredResult := result
	if !verbose {
		filteredResult = &types.LintResult{
			LintExpressions: filterInfoMessages(result.LintExpressions),
			IsComplete:      result.IsComplete,
		}
	}

	switch format {
	case "json":
		return printLintJSON(filteredResult)
	case "yaml":
		return printLintYAML(filteredResult)
	case "table", "":
		return printLintTable(filteredResult)
	default:
		return errors.Errorf("unsupported output format: %s (supported: table, json, yaml)", format)
	}
}

func filterInfoMessages(expressions []types.LintExpression) []types.LintExpression {
	filtered := []types.LintExpression{}
	for _, expr := range expressions {
		if expr.Type != "info" {
			filtered = append(filtered, expr)
		}
	}
	return filtered
}

func printLintJSON(result *types.LintResult) error {
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal JSON")
	}
	fmt.Println(string(output))
	return nil
}

func printLintYAML(result *types.LintResult) error {
	output, err := yaml.Marshal(result)
	if err != nil {
		return errors.Wrap(err, "failed to marshal YAML")
	}
	fmt.Println(string(output))
	return nil
}

func printLintTable(result *types.LintResult) error {
	if len(result.LintExpressions) == 0 {
		fmt.Println("No issues found")
		return nil
	}

	// Print header
	fmt.Printf("%-30s %-8s %-40s %s\n", "RULE", "TYPE", "PATH", "MESSAGE")
	fmt.Println(strings.Repeat("-", 120))

	// Print each expression
	for _, expr := range result.LintExpressions {
		path := expr.Path
		if len(path) > 40 {
			path = "..." + path[len(path)-37:]
		}

		message := expr.Message
		if len(message) > 50 {
			message = message[:47] + "..."
		}

		lineInfo := ""
		if len(expr.Positions) > 0 && expr.Positions[0].Start.Line > 0 {
			lineInfo = fmt.Sprintf(":%d", expr.Positions[0].Start.Line)
		}

		fmt.Printf("%-30s %-8s %-40s %s%s\n",
			expr.Rule,
			expr.Type,
			path+lineInfo,
			message,
			"",
		)
	}

	fmt.Println()

	// Count info messages
	infoCount := 0
	for _, expr := range result.LintExpressions {
		if expr.Type == "info" {
			infoCount++
		}
	}

	// Show summary based on what's present
	if infoCount > 0 {
		fmt.Printf("Summary: %d error(s), %d warning(s), %d info\n", result.ErrorCount(), result.WarningCount(), infoCount)
	} else {
		fmt.Printf("Summary: %d error(s), %d warning(s)\n", result.ErrorCount(), result.WarningCount())
	}

	return nil
}
