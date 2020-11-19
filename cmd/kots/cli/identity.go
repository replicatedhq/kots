package cli

import (
	"io/ioutil"

	ghodssyaml "github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/identity"
	identitytypes "github.com/replicatedhq/kots/pkg/identity/types"
	ingress "github.com/replicatedhq/kots/pkg/ingress"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func IdentityServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "identity-service",
		Short:  "KOTS identity service",
		Hidden: true,
	}

	cmd.AddCommand(IdentityServiceInstallCmd())
	cmd.AddCommand(IdentityServiceEnableSharedPasswordCmd())

	return cmd
}

func IdentityServiceInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "install",
		Short:         "Install the KOTS identity service",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			log := logger.NewLogger()

			clientset, err := k8sutil.GetClientset(kubernetesConfigFlags)
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			ingressConfig, err := ingress.GetConfig(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get ingress config")
			}
			if ingressConfig == nil {
				// TODO: a cli command to enable ingress
				return errors.New("ingress is not enabled")
			}

			identityConfig := identitytypes.Config{}
			if identityConfigPath := v.GetString("identity-config"); identityConfigPath != "" {
				content, err := ioutil.ReadFile(identityConfigPath)
				if err != nil {
					return errors.Wrap(err, "failed to read identity service connectors file")
				}
				if err := ghodssyaml.Unmarshal(content, &identityConfig); err != nil {
					return errors.Wrap(err, "failed to unmarshal identity service connectors yaml")
				}
			}

			log.ChildActionWithSpinner("Deploying the Identity Service")

			identityConfig.DisablePasswordAuth = true

			if err := identity.SetConfig(cmd.Context(), namespace, identityConfig); err != nil {
				return errors.Wrap(err, "failed to set identity config")
			}

			if err := identity.Deploy(cmd.Context(), log, clientset, namespace, identityConfig, *ingressConfig); err != nil {
				return errors.Wrap(err, "failed to deploy identity service")
			}

			log.FinishSpinner()

			return nil
		},
	}

	cmd.Flags().String("identity-config", "", "path to a yaml file containing the KOTS identity service configuration")

	return cmd
}

func IdentityServiceEnableSharedPasswordCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "enable-shared-password",
		Short:         "Enable shared password",
		Long:          "Enable the shared password login option in the KOTS Admin Console.",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			log := logger.NewLogger()

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			log.ChildActionWithSpinner("Updating the Identity Service configuration")

			identityConfig, err := identity.GetConfig(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get identity config")
			}

			identityConfig.DisablePasswordAuth = false

			if err := identity.SetConfig(cmd.Context(), namespace, *identityConfig); err != nil {
				return errors.Wrap(err, "failed to set identity config")
			}

			log.FinishSpinner()

			return nil
		},
	}

	return cmd
}
