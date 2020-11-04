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
		Short:         "List available instance backups",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			options := snapshot.ListInstanceBackupsOptions{
				Namespace: v.GetString("namespace"),
			}
			backups, err := snapshot.ListInstanceBackups(options)
			if err != nil {
				return errors.Wrap(err, "failed to list instance backups")
			}

			print.Backups(backups)

			return nil
		},
	}

	cmd.Flags().StringP("namespace", "n", "", "filter by the namespace in which kots/kotsadm is installed")

	return cmd
}
