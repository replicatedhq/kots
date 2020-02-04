package cli

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
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

			upgradeOptions := kotsadmtypes.UpgradeOptions{
				Namespace:  v.GetString("namespace"),
				Kubeconfig: v.GetString("kubeconfig"),
			}

			kotsadm.OverrideVersion = v.GetString("kotsadm-tag")
			kotsadm.OverrideRegistry = v.GetString("kotsadm-registry")
			kotsadm.OverrideNamespace = v.GetString("kotsadm-namespace")

			log := logger.NewLogger()

			if upgradeOptions.Namespace != "default" {
				log.ActionWithoutSpinner("Upgrading Admin Console")
			} else {
				log.ActionWithoutSpinner("Upgrading Admin Console in the default namespace")
			}
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

	cmd.Flags().String("kotsadm-tag", "", "set to override the tag of kotsadm. this may create an incompatible deployment because the version of kots and kotsadm are designed to work together")
	cmd.Flags().String("kotsadm-registry", "", "set to override the registry of kotsadm image. this may create an incompatible deployment because the version of kots and kotsadm are designed to work together")
	cmd.Flags().String("kotsadm-namespace", "", "set to override the namespace of kotsadm image. this may create an incompatible deployment because the version of kots and kotsadm are designed to work together")
	cmd.Flags().MarkHidden("kotsadm-tag")
	cmd.Flags().MarkHidden("kotsadm-registry")
	cmd.Flags().MarkHidden("kotsadm-namespace")

	return cmd
}
