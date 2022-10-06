package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func PluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "plugin",
		Short:         "Run commands for helm plugin",
		Long:          `Run commands for helm plugin, this serves as an entry point for subcommands such as preflight`,
		SilenceUsage:  false,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	cmd.AddCommand(PreflightCmd())
	return cmd
}
