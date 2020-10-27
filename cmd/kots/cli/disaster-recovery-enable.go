package cli

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func DisasterRecoveryEnableCmd() *cobra.Command {
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

			options := &types.EnableDisasterRecoveryOptions{
				Namespace:             v.GetString("namespace"),
				KubernetesConfigFlags: kubernetesConfigFlags,
			}
			if err := kotsadm.EnableDisasterRecovery(options); err != nil {
				return errors.Cause(err)
			}

			log := logger.NewLogger()
			log.ActionWithoutSpinner("Disaster recovery has been enabled successfully.") // TODO: mention apps as well

			return nil
		},
	}

	cmd.Flags().StringP("namespace", "n", "default", "namespace in which kots/kotsadm is installed")

	return cmd
}
