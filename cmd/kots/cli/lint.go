package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/lint"
	"github.com/replicatedhq/kots/pkg/lint/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

func LintCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint [path]",
		Short: "Lint a KOTS application",
		Long: `Lint validates a KOTS application for errors and warnings.

The path can be:
  - A directory containing YAML files
  - A tar archive
  - If omitted, uses current directory

This is a development/test CLI. Production usage will be via 'replicated kots lint'.`,
		Args: cobra.MaximumNArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()
			log := logger.NewCLILogger(cmd.OutOrStdout())

			// Get path argument
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			// Get flags
			outputFormat := v.GetString("output")
			failOnWarn := v.GetBool("fail-on-warn")
			offline := v.GetBool("offline")
			verbose := v.GetBool("verbose")

			// Initialize OPA
			log.Info("Initializing linter...")
			if err := lint.InitOPA(); err != nil {
				return errors.Wrap(err, "failed to initialize OPA")
			}

			// Load files
			log.ActionWithSpinner("Loading files from %s", path)
			files, err := lint.LoadFiles(path)
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrapf(err, "failed to load files from %s", path)
			}
			log.FinishSpinner()

			log.Info("Found %d files to lint", len(files))

			// Run linter
			log.ActionWithSpinner("Running linter...")
			result, err := lint.LintSpecFiles(cmd.Context(), files, lint.LintOptions{
				SkipNetworkChecks: offline,
				Verbose:           verbose,
			})
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to lint files")
			}
			log.FinishSpinner()

			// Format output
			if err := printLintResult(result, outputFormat, verbose); err != nil {
				return errors.Wrap(err, "failed to print result")
			}

			// Exit code
			if result.HasErrors() {
				log.Error(errors.Errorf("linting failed with %d error(s)", result.ErrorCount()))
				os.Exit(1)
			}
			if failOnWarn && result.HasWarnings() {
				log.Error(errors.Errorf("linting failed with %d warning(s)", result.WarningCount()))
				os.Exit(1)
			}

			if result.IsComplete {
				log.Info("✓ All checks passed!")
			} else {
				log.Info("Linting stopped early due to errors")
			}

			return nil
		},
	}

	cmd.Flags().StringP("output", "o", "table", "Output format (table, json, yaml)")
	cmd.Flags().Bool("fail-on-warn", false, "Exit with error on warnings")
	cmd.Flags().Bool("offline", false, "Skip checks that require network (e.g., version validation)")
	cmd.Flags().BoolP("verbose", "v", false, "Show info-level messages (default: only show warnings and errors)")

	return cmd
}

func printLintResult(result *types.LintResult, format string, verbose bool) error {
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
		return printJSON(filteredResult)
	case "yaml":
		return printYAML(filteredResult)
	case "table", "":
		return printTable(filteredResult)
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

func printJSON(result *types.LintResult) error {
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal JSON")
	}
	fmt.Println(string(output))
	return nil
}

func printYAML(result *types.LintResult) error {
	output, err := yaml.Marshal(result)
	if err != nil {
		return errors.Wrap(err, "failed to marshal YAML")
	}
	fmt.Println(string(output))
	return nil
}

func printTable(result *types.LintResult) error {
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
