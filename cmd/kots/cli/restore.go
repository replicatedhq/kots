package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "restore",
		Short:         "Provides wrapper functionality to interface with the restore source",
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

	cmd.AddCommand(RestoreCreateCmd())
	cmd.AddCommand(RestoreListCmd())

	return cmd
}
