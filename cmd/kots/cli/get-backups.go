package cli

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/print"
	"github.com/replicatedhq/kots/pkg/snapshot"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func GetBackupsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "backups",
		Aliases:       []string{"backup"},
		Short:         "Get backups",
		Long:          "",
		SilenceUsage:  false,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: getBackupsCmd,
	}

	cmd.Flags().StringP("output", "o", "", "output format. supported values: json")

	return cmd
}

func getBackupsCmd(cmd *cobra.Command, args []string) error {
	v := viper.GetViper()

	options := snapshot.ListInstanceBackupsOptions{
		Namespace: v.GetString("namespace"),
	}
	backups, err := snapshot.ListInstanceBackups(cmd.Context(), options)
	if err != nil {
		return errors.Wrap(err, "failed to list instance backups")
	}

	print.Backups(backups, v.GetString("output"))

	return nil
}
