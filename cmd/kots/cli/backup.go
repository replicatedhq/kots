package cli

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/snapshot"
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
			v := viper.GetViper()

			namespace := v.GetString("namespace")

			options := snapshot.CreateInstanceBackupOptions{
				Namespace:             namespace,
				KubernetesConfigFlags: kubernetesConfigFlags,
				Wait:                  v.GetBool("wait"),
			}
			if err := snapshot.CreateInstanceBackup(options); err != nil {
				return errors.Wrap(err, "failed to create instance backup")
			}

			return nil
		},
	}

	cmd.Flags().StringP("namespace", "n", "default", "namespace in which kots/kotsadm is installed")
	cmd.Flags().Bool("wait", true, "wait for the backup to finish")

	cmd.AddCommand(BackupListCmd())

	return cmd
}
