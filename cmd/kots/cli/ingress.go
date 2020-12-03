package cli

import (
	"io/ioutil"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/identity"
	"github.com/replicatedhq/kots/pkg/ingress"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/kotsutil"
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

			ingressConfig := kotsv1beta1.IngressConfig{}
			if ingressConfigPath := v.GetString("ingress-config"); ingressConfigPath != "" {
				content, err := ioutil.ReadFile(ingressConfigPath)
				if err != nil {
					return errors.Wrap(err, "failed to read ingress service config file")
				}

				s, err := kotsutil.LoadIngressConfigFromContents(content)
				if err != nil {
					return errors.Wrap(err, "failed to decoce ingress service config")
				}
				ingressConfig = *s
			}

			log.ChildActionWithSpinner("Enabling ingress for the Admin Console")

			ingressConfig.Spec.Enabled = true

			if err := ingress.SetConfig(cmd.Context(), namespace, ingressConfig); err != nil {
				return errors.Wrap(err, "failed to set identity config")
			}

			if err := kotsadm.EnsureIngress(cmd.Context(), namespace, clientset, ingressConfig.Spec); err != nil {
				return errors.Wrap(err, "failed to ensure ingress")
			}

			log.FinishSpinner()

			identityConfig, err := identity.GetConfig(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get identity config")
			}

			if identityConfig.Spec.Enabled {
				log.ChildActionWithSpinner("Configuring the Identity Service")

				if err := identity.ConfigValidate(identityConfig.Spec, ingressConfig.Spec); err != nil {
					return errors.Wrap(err, "failed to validate identity config")
				}

				// we have to update the dex secret if kotsadm ingress is changing because it relies on the redirect uri
				if err := identity.Configure(cmd.Context(), clientset, namespace, *identityConfig, ingressConfig); err != nil {
					return errors.Wrap(err, "failed to patch identity service")
				}

				log.FinishSpinner()
			}

			return nil
		},
	}

	cmd.Flags().String("ingress-config", "", "path to a kots.Ingress resource file")

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

			ingressConfig.Spec.Enabled = false

			identityConfig, err := identity.GetConfig(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get identity config")
			}

			if err := identity.ConfigValidate(identityConfig.Spec, ingressConfig.Spec); err != nil {
				return errors.Wrap(err, "failed to validate identity config")
			}

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

	cmd.Flags().String("ingress-config", "", "path to a kots.Ingress resource file")

	return cmd
}
