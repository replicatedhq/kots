package cli

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/snapshot"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RestoreCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "create [backup name]",
		Short:         "Starts an instance restore from a backup",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				cmd.Help()
				os.Exit(1)
			}

			instanceRestoreOptions := snapshot.InstanceRestoreOptions{
				BackupName: args[0],
			}
			restore, err := snapshot.InstanceRestore(instanceRestoreOptions)
			if err != nil {
				return errors.Cause(err)
			}

			log := logger.NewLogger()
			log.ActionWithoutSpinner(fmt.Sprintf("Restore request has been created. Restore name is %s", restore.ObjectMeta.Name))

			return nil
		},
	}

	return cmd
}
