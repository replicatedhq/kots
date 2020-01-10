package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func UpstreamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "upstream",
		Short:         "Provides wrapper functionality to interface with the upstream source",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				cmd.Help()
				os.Exit(1)
			}

			return nil
		},
	}

	cmd.AddCommand(UpstreamUpgradeCmd())

	return cmd
}
