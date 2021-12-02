package cli

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/print"
	"github.com/replicatedhq/kots/pkg/snapshot"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func GetRestoresCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "restores",
		Aliases:       []string{"restore"},
		Short:         "Get restores",
		Long:          "",
		SilenceUsage:  false,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: getRestoresCmd,
	}

	cmd.Flags().StringP("output", "o", "", "output format. supported values: json")

	return cmd
}

func getRestoresCmd(cmd *cobra.Command, args []string) error {
	v := viper.GetViper()

	options := snapshot.ListInstanceRestoresOptions{
		Namespace: v.GetString("namespace"),
	}
	restores, err := snapshot.ListInstanceRestores(cmd.Context(), options)
	if err != nil {
		return errors.Wrap(err, "failed to list instance restores")
	}

	print.Restores(restores, v.GetString("output"))

	return nil
}
