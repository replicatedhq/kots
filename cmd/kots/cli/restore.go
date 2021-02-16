package cli

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/print"
	"github.com/replicatedhq/kots/pkg/snapshot"
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
			v := viper.GetViper()

			backupName := v.GetString("from-backup")
			if backupName == "" {
				fmt.Printf("a backup name must be provided via the '--from-backup' flag\n")
				os.Exit(1)
			}

			options := snapshot.RestoreInstanceBackupOptions{
				BackupName:            backupName,
				KubernetesConfigFlags: kubernetesConfigFlags,
				WaitForApps:           v.GetBool("wait-for-apps"),
				VeleroNamespace:       v.GetString("velero-namespace"),
			}
			_, err := snapshot.RestoreInstanceBackup(options)
			if err != nil {
				return errors.Wrap(err, "failed to restore instance backup")
			}

			return nil
		},
	}

	cmd.Flags().String("from-backup", "", "the name of the backup to restore from")
	cmd.Flags().String("velero-namespace", "", "namespace in which velero is installed")
	cmd.Flags().Bool("wait-for-apps", true, "wait for all applications to be restored")

	cmd.AddCommand(RestoreListCmd())

	return cmd
}

func RestoreListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ls",
		Short:         `List available restores (this command is deprecated, please use "kubectl kots get restores" instead)`,
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			options := snapshot.ListInstanceRestoresOptions{
				Namespace: v.GetString("namespace"),
			}
			restores, err := snapshot.ListInstanceRestores(options)
			if err != nil {
				return errors.Wrap(err, "failed to list instance restores")
			}

			print.Restores(restores)

			return nil
		},
	}

	cmd.Flags().StringP("namespace", "n", "", "filter by the namespace in which kots/kotsadm is installed")

	return cmd
}
