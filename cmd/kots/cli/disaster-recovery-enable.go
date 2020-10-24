package cli

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func UpstreamUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "enable",
		Short:         "Enables the disaster recovery feature",
		Long:          "",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			disasterRecoveryOptions := &types.DisasterRecoveryOptions{
				Namespace: v.GetString("namespace"),
			}
			if err := kotsadm.EnableDisasterRecovery(disasterRecoveryOptions); err != nil {
				return errors.Cause(err)
			}

			return nil
		},
	}

	cmd.Flags().StringP("namespace", "n", "default", "namespace in which kots/kotsadm is installed")

	return cmd
}
