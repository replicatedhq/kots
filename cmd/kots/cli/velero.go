package cli

import (
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation"

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

			veleroNamespace := v.GetString("velero-namespace")
			if err := validateVeleroNamespace(veleroNamespace); err != nil {
				return err
			}

			kotsadmNamespace := v.GetString("namespace")
			if err := validateNamespace(kotsadmNamespace); err != nil {
				return err
			}

			if err := snapshot.EnsureVeleroPermissions(veleroNamespace, kotsadmNamespace); err != nil {
				return err
			}

			if err := snapshot.EnsureVeleroNamespaceConfigMap(veleroNamespace, kotsadmNamespace); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringP("namespace", "n", "", "namespace in which kots/kotsadm is installed")
	cmd.Flags().String("velero-namespace", "", "namespace in which velero is installed")

	return cmd
}

func validateVeleroNamespace(namespace string) error {
	if namespace == "" {
		return errors.New("velero-namespace is required")
	}
	if strings.Contains(namespace, "_") {
		return errors.New("velero-namespace should not contain the _ character")
	}

	errs := validation.IsValidLabelValue(namespace)
	if len(errs) > 0 {
		return errors.New(errs[0])
	}

	return nil
}
