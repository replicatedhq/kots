package cli

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/print"
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

			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}

			output := v.GetString("output")
			if output != "json" && output != "" {
				return errors.Errorf("output format %s not supported (allowed formats are: json)", output)
			}

			options := snapshot.CreateInstanceBackupOptions{
				Namespace: namespace,
				Wait:      v.GetBool("wait"),
				Silent:    output != "",
			}
			backupRes, err := snapshot.CreateInstanceBackup(cmd.Context(), options)
			if err != nil && output == "" {
				return errors.Wrap(err, "failed to create instance backup")
			} else if err != nil {
				backupRes = &snapshot.BackupResponse{
					Error: fmt.Sprint(errors.Wrap(err, "failed to create instance backup")),
				}
			}

			log := logger.NewCLILogger(cmd.OutOrStdout())
			if output == "json" {
				outputJSON, err := json.Marshal(backupRes)
				if err != nil {
					return errors.Wrap(err, "error marshaling JSON")
				}
				log.Info("%s", string(outputJSON))
			}

			return nil
		},
	}

	cmd.Flags().StringP("namespace", "n", "default", "namespace in which kots/kotsadm is installed")
	cmd.Flags().Bool("wait", true, "wait for the backup to finish")
	cmd.Flags().StringP("output", "o", "", "output format (currently supported: json)")

	cmd.AddCommand(BackupListCmd())

	return cmd
}

func BackupListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ls",
		Short:         `List available instance backups (this command is deprecated, please use "kubectl kots get backups" instead)`,
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}

			options := snapshot.ListInstanceBackupsOptions{
				Namespace: namespace,
			}
			backups, err := snapshot.ListInstanceBackups(cmd.Context(), options)
			if err != nil {
				return errors.Wrap(err, "failed to list instance backups")
			}

			print.Backups(backups, "")

			return nil
		},
	}

	cmd.Flags().StringP("namespace", "n", "", "filter by the namespace in which kots/kotsadm is installed")

	return cmd
}
