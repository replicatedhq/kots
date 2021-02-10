package cli

import (
	"github.com/replicatedhq/kots/pkg/snapshot"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func VeleroCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "velero",
		Short: "KOTS Velero interface",
	}

	cmd.AddCommand(EnsurePermissionsCmd())

	return cmd
}

func EnsurePermissionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ensure-permissions",
		Short:         "Ensures the necessary permissions that enables the Admin Console to access Velero.",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			if err := snapshot.EnsureVeleroPermissions(namespace); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringP("namespace", "n", "", "namespace in which kots/kotsadm is installed")

	return cmd
}
