package cli

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/snapshot"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RestoreCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "create --from-backup [backup name]",
		Short:         "Starts an instance restore from a backup",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			backupName := v.GetString("from-backup")
			if backupName == "" {
				fmt.Printf("a backup name must be provided via the '--from-backup' flag\n")
				os.Exit(1)
			}

			options := snapshot.CreateInstanceRestoreOptions{
				BackupName:            backupName,
				KubernetesConfigFlags: kubernetesConfigFlags,
			}
			_, err := snapshot.CreateInstanceRestore(options)
			if err != nil {
				return errors.Wrap(err, "failed to create instance restore")
			}

			return nil
		},
	}

	cmd.Flags().String("from-backup", "", "the name of the backup to restore from")

	return cmd
}
