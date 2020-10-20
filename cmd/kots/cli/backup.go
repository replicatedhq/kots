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
		Short:         "Create an instance backup/snapshot",
		Long:          `Create an instance backup/snapshot`,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			instanceBackupOptions := snapshot.InstanceBackupOptions{
				Namespace:             v.GetString("namespace"),
				KubernetesConfigFlags: kubernetesConfigFlags,
			}
			if err := snapshot.InstanceBackup(instanceBackupOptions); err != nil {
				return errors.Cause(err)
			}

			return nil
		},
	}

	return cmd
}
