package cli

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/print"
	"github.com/replicatedhq/kots/pkg/snapshot"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func BackupListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ls",
		Short:         "List available backups",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			backups, err := snapshot.ListBackups()
			if err != nil {
				return errors.Cause(err)
			}

			print.Backups(backups)

			return nil
		},
	}

	return cmd
}
