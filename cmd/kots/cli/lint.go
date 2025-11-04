package cli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/lint"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/print"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		Hidden: true, // Hidden during migration to replicated CLI
		Args:   cobra.MaximumNArgs(1),
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

			// Silence loggers for structured output formats to prevent pollution
			if outputFormat == "json" || outputFormat == "yaml" {
				logrus.SetLevel(logrus.PanicLevel) // Silence logrus (used by lint package)
				logger.SetSilent()                 // Silence zap logger
				log.Silence()                      // Silence CLI logger
			}

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
			if err := print.LintResult(result, outputFormat, verbose); err != nil {
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
