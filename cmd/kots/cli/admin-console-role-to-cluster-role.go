package cli

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func AdminConsoleAllowNamespace() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "allow-namespace",
		Short: "Grant the Admin Console RBAC policies to access resources in additional namespaces",
		Long: `This command will convert the K8s rbac policies to clusterroles, allowing access to other namespaces.
This is useful when using snapshots on existing clusters when velero is install to a separate namespace`,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			log := logger.NewLogger()

			log.ActionWithSpinner("Reconciling Kubernetes RBAC policies")
			err := kotsadm.EnsureAdditionalNamespaces(log, v.GetStringSlice("additional-namespace"), v.GetString("namespace"), v.GetBool("prune"))
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to find kotsadm pod")
			}
			log.FinishSpinner()

			log.ActionWithoutSpinner("")
			log.ActionWithoutSpinner("The Admin Console role has been updated to include permissions in the additional namespaces")
			log.ActionWithoutSpinner("To access the Admin Console, run kubectl kots admin-console --namespace %s", v.GetString("namespace"))
			log.ActionWithoutSpinner("")

			return nil
		},
	}

	cmd.Flags().String("kubeconfig", defaultKubeConfig(), "the kubeconfig to use")
	cmd.Flags().StringP("namespace", "n", "default", "the namespace where the admin console is running")
	cmd.Flags().Bool("prune", false, "when set, this will prune unused namespaces from the role/clusterrole")

	cmd.Flags().StringSlice("additional-namespace", []string{}, "the list of additional namespaces to grant permissions to")

	return cmd
}
