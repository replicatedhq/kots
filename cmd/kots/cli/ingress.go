package cli

import (
	"io/ioutil"

	ghodssyaml "github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/identity"
	"github.com/replicatedhq/kots/pkg/ingress"
	ingresstypes "github.com/replicatedhq/kots/pkg/ingress/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func IngressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "ingress",
		Short:  "KOTS ingress",
		Hidden: true,
	}

	cmd.AddCommand(IngressInstallCmd())
	cmd.AddCommand(IngressUninstallCmd())

	return cmd
}

func IngressInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "install",
		Short:         "Install Ingress for the KOTS Admin Console",
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

			ingressConfig := ingresstypes.Config{}
			if ingressConfigPath := v.GetString("ingress-config"); ingressConfigPath != "" {
				content, err := ioutil.ReadFile(ingressConfigPath)
				if err != nil {
					return errors.Wrap(err, "failed to read ingress config file")
				}
				if err := ghodssyaml.Unmarshal(content, &ingressConfig); err != nil {
					return errors.Wrap(err, "failed to unmarshal ingress config yaml")
				}
			}

			log.ChildActionWithSpinner("Enabling ingress for the Admin Console")

			ingressConfig.Enabled = true

			if err := ingress.SetConfig(cmd.Context(), namespace, ingressConfig); err != nil {
				return errors.Wrap(err, "failed to set identity config")
			}

			if err := kotsadm.EnsureIngress(cmd.Context(), namespace, clientset, ingressConfig); err != nil {
				return errors.Wrap(err, "failed to ensure ingress")
			}

			log.FinishSpinner()

			identityConfig, err := identity.GetConfig(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get identity config")
			}

			if identityConfig.Enabled {
				log.ChildActionWithSpinner("Deploying the Identity Service")

				// we have to re-deploy the identity service if kotsadm ingress is changing
				if err := identity.Deploy(cmd.Context(), log, clientset, namespace, *identityConfig, ingressConfig); err != nil {
					return errors.Wrap(err, "failed to deploy identity service")
				}

				log.FinishSpinner()
			}

			return nil
		},
	}

	cmd.Flags().String("ingress-config", "", "path to a yaml file containing the KOTS ingress configuration")

	return cmd
}

func IngressUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "uninstall",
		Short:         "Uninstall Ingress for the KOTS Admin Console",
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

			log.ChildActionWithSpinner("Updating the Admin Console ingress configuration")

			ingressConfig, err := ingress.GetConfig(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get ingress config")
			}

			ingressConfig.Enabled = false

			if err := ingress.SetConfig(cmd.Context(), namespace, *ingressConfig); err != nil {
				return errors.Wrap(err, "failed to set ingress config")
			}

			log.FinishSpinner()

			log.ChildActionWithSpinner("Uninstalling ingress for the Admin Console")

			if err := kotsadm.DeleteIngress(cmd.Context(), namespace, clientset); err != nil {
				return errors.Wrap(err, "failed to uninstall ingress")
			}

			log.FinishSpinner()

			return nil
		},
	}

	cmd.Flags().String("ingress-config", "", "path to a yaml file containing the KOTS ingress configuration")

	return cmd
}
