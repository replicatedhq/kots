package cli

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func AdminConsoleUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "upgrade",
		Short:         "Upgrade the admin console to the latest version",
		Long:          "Upgrade the admin console to the latest version",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			upgradeOptions := kotsadm.UpgradeOptions{
				Namespace:  v.GetString("namespace"),
				Kubeconfig: v.GetString("kubeconfig"),
			}

			log := logger.NewLogger()
			log.ActionWithoutSpinner("Upgrading Admin Console")
			if err := kotsadm.Upgrade(upgradeOptions); err != nil {
				return errors.Wrap(err, "failed to upgrade")
			}

			log.ActionWithoutSpinner("")
			log.ActionWithoutSpinner("The Admin Console is running the latest version")
			log.ActionWithoutSpinner("To access the Admin Console, run kubectl kots admin-console --namespace %s", v.GetString("namespace"))
			log.ActionWithoutSpinner("")

			return nil
		},
	}

	cmd.Flags().String("kubeconfig", defaultKubeConfig(), "the kubeconfig to use")
	cmd.Flags().StringP("namespace", "n", "default", "the namespace where the admin console is running")

	return cmd
}
