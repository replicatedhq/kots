package cli

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/snapshot"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "restore",
		Short:         "Restore an instance from a backup",
		Long:          `Restore an instance from a backup`,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			instanceRestoreOptions := snapshot.InstanceRestoreOptions{
				BackupName: v.GetString("from-backup"),
			}
			if err := snapshot.InstanceRestore(instanceRestoreOptions); err != nil {
				return errors.Cause(err)
			}

			return nil
		},
	}

	cmd.Flags().String("from-backup", "", "the backup to create the restore from")

	return cmd
}
