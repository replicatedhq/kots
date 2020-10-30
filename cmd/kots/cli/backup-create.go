package cli

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/snapshot"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func BackupCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "create",
		Short:         "Creates an instance backup/snapshot",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			options := snapshot.CreateInstanceBackupOptions{
				Namespace:             v.GetString("namespace"),
				KubernetesConfigFlags: kubernetesConfigFlags,
			}
			if err := snapshot.CreateInstanceBackup(options); err != nil {
				return errors.Wrap(err, "failed to create instance backup")
			}

			return nil
		},
	}

	cmd.Flags().StringP("namespace", "n", "default", "namespace in which kots/kotsadm is installed")

	return cmd
}
