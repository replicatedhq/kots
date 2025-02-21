package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/print"
	"github.com/replicatedhq/kots/pkg/snapshot"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type RestoreOutput struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

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

			output := v.GetString("output")
			if output != "json" && output != "" {
				return errors.Errorf("output format %s not supported (allowed formats are: json)", output)
			}

			if v.GetBool("exclude-admin-console") && v.GetBool("exclude-apps") {
				return errors.New("--exclude-admin-console and --exclude-apps cannot be used together")
			}

			var restoreOutput RestoreOutput
			options := snapshot.RestoreInstanceBackupOptions{
				BackupName:          backupName,
				ExcludeAdminConsole: v.GetBool("exclude-admin-console"),
				ExcludeApps:         v.GetBool("exclude-apps"),
				WaitForApps:         v.GetBool("wait-for-apps"),
				VeleroNamespace:     v.GetString("velero-namespace"),
				Silent:              output != "",
			}
			err := snapshot.RestoreInstanceBackup(cmd.Context(), options)
			if err != nil && output == "" {
				return errors.Wrap(err, "failed to restore instance backup")
			} else if err != nil {
				restoreOutput.Error = fmt.Sprint(errors.Wrap(err, "failed to restore instance backup"))
			} else {
				restoreOutput.Success = true
			}

			log := logger.NewCLILogger(cmd.OutOrStdout())
			if output == "json" {
				outputJSON, err := json.Marshal(restoreOutput)
				if err != nil {
					return errors.Wrap(err, "error marshaling JSON")
				}
				log.Info("%s", string(outputJSON))
			}

			return nil
		},
	}

	cmd.Flags().String("from-backup", "", "the name of the backup to restore from")
	cmd.Flags().String("velero-namespace", "", "namespace in which velero is installed")
	cmd.Flags().Bool("exclude-admin-console", false, "exclude restoring the admin console and only restore the application(s)")
	cmd.Flags().Bool("exclude-apps", false, "exclude restoring the application(s) and only restore the admin console")
	cmd.Flags().Bool("wait-for-apps", true, "wait for all applications to be restored")
	cmd.Flags().StringP("output", "o", "", "output format (currently supported: json)")

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

			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}
			options := snapshot.ListInstanceRestoresOptions{
				Namespace: namespace,
			}
			restores, err := snapshot.ListInstanceRestores(cmd.Context(), options)
			if err != nil {
				return errors.Wrap(err, "failed to list instance restores")
			}

			print.Restores(restores, "")

			return nil
		},
	}

	cmd.Flags().StringP("namespace", "n", "", "filter by the namespace in which kots/kotsadm is installed")

	return cmd
}
