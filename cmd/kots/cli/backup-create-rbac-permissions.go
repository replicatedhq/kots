package cli

import (
	"github.com/replicatedhq/kots/pkg/snapshot"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func BackupCreateRBACPermissionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "create-rbac-permissions",
		Short:         "Creates the necessary minimal RBAC permissions that enables the Admin Console to access Velero.",
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

			if err := snapshot.CreateRBACPermissions(namespace); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().String("namespace", "n", "namespace in which kots/kotsadm is installed")

	return cmd
}
