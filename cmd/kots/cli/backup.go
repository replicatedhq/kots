package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func BackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "backup",
		Short:         "Provides wrapper functionality to interface with the backup source",
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

	cmd.AddCommand(BackupCreateCmd())
	cmd.AddCommand(BackupListCmd())

	return cmd
}
