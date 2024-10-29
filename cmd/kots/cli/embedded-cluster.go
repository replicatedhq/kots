package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func EmbeddedClusterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "embedded-cluster",
		Short:                 "Configure embedded-cluster resources",
		Long:                  ``,
		Hidden:                true,
		DisableFlagsInUseLine: true, // removes "kots set [flags]" from usage in help output

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

	cmd.AddCommand(EmbeddedClusterConfirmManagementCmd())

	return cmd
}
