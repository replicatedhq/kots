package cli

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/identity"
	ingress "github.com/replicatedhq/kots/pkg/ingress"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
)

func IdentityServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "identity-service",
		Short:  "KOTS Identity Service",
		Hidden: true,
	}

	cmd.AddCommand(IdentityServiceInstallCmd())
	cmd.AddCommand(IdentityServiceConfigureCmd())
	cmd.AddCommand(IdentityServiceUninstallCmd())
	cmd.AddCommand(IdentityServiceEnableSharedPasswordCmd())
	cmd.AddCommand(IdentityServiceOIDCCallbackURLCmd())

	return cmd
}

func IdentityServiceInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "install",
		Short:         "Install the KOTS Identity Service",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			log := logger.NewCLILogger(cmd.OutOrStdout())

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}

			ingressConfig, err := ingress.GetConfig(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get ingress config")
			}

			identityConfig := kotsv1beta1.IdentityConfig{}
			if identityConfigPath := v.GetString("identity-config"); identityConfigPath != "" {
				content, err := ioutil.ReadFile(identityConfigPath)
				if err != nil {
					return errors.Wrap(err, "failed to read identity service config file")
				}

				s, err := kotsutil.LoadIdentityConfigFromContents(content)
				if err != nil {
					return errors.Wrap(err, "failed to decoce identity service config")
				}
				identityConfig = *s
			}

			registryConfig, err := getRegistryConfig(v, clientset, "")
			if err != nil {
				return errors.Wrap(err, "failed to get registry config")
			}

			proxyEnv := getHttpProxyEnv(v)

			return identityServiceDeploy(cmd.Context(), log, clientset, namespace, identityConfig, *ingressConfig, registryConfig, proxyEnv, v.GetBool("apply-app-branding"))
		},
	}

	cmd.Flags().String("identity-config", "", "path to a manifest containing the KOTS identity service configuration (must be apiVersion: kots.io/v1beta1, kind: IdentityConfig)")
	cmd.Flags().Bool("airgap", false, "set to true to run install in airgapped mode. setting --airgap-bundle implies --airgap=true.")
	cmd.Flags().String("http-proxy", "", "sets HTTP_PROXY environment variable in KOTS Identity Service components")
	cmd.Flags().String("https-proxy", "", "sets HTTPS_PROXY environment variable in KOTS Identity Service components")
	cmd.Flags().String("no-proxy", "", "sets NO_PROXY environment variable in KOTS Identity Service components")
	cmd.Flags().Bool("copy-proxy-env", false, "copy proxy environment variables from current environment into KOTS Identity Service components")
	cmd.Flags().Bool("apply-app-branding", false, "apply app branding to the identity login screen")

	registryFlags(cmd.Flags())

	return cmd
}

func IdentityServiceConfigureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "configure",
		Short:         "Configure the KOTS Identity Service",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			log := logger.NewCLILogger(cmd.OutOrStdout())

			clientset, err := k8sutil.GetClientset()
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

			identityConfig := kotsv1beta1.IdentityConfig{}
			if identityConfigPath := v.GetString("identity-config"); identityConfigPath != "" {
				content, err := ioutil.ReadFile(identityConfigPath)
				if err != nil {
					return errors.Wrap(err, "failed to read identity service config file")
				}

				s, err := kotsutil.LoadIdentityConfigFromContents(content)
				if err != nil {
					return errors.Wrap(err, "failed to decoce identity service config")
				}
				identityConfig = *s
			}

			proxyEnv := getHttpProxyEnv(v)
			return identityServiceConfigure(cmd.Context(), log, clientset, namespace, identityConfig, *ingressConfig, proxyEnv, v.GetBool("apply-app-branding"))
		},
	}

	cmd.Flags().String("identity-config", "", "path to a manifest containing the KOTS identity service configuration (must be apiVersion: kots.io/v1beta1, kind: IdentityConfig)")
	cmd.Flags().String("http-proxy", "", "sets HTTP_PROXY environment variable in KOTS Identity Service components")
	cmd.Flags().String("https-proxy", "", "sets HTTPS_PROXY environment variable in KOTS Identity Service components")
	cmd.Flags().String("no-proxy", "", "sets NO_PROXY environment variable in KOTS Identity Service components")
	cmd.Flags().Bool("copy-proxy-env", false, "copy proxy environment variables from current environment into KOTS Identity Service components")
	cmd.Flags().Bool("apply-app-branding", false, "apply app branding to the identity login screen")

	return cmd
}

func IdentityServiceUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "uninstall",
		Short:         "Uninstall the KOTS identity service",
		Long:          "Uninstall the KOTS identity service. This will re-enable shared password authentication.",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			log := logger.NewCLILogger(cmd.OutOrStdout())

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			log.ChildActionWithSpinner("Updating the Identity Service configuration")

			identityConfig, err := identity.GetConfig(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get identity config")
			}

			identityConfig.Spec.Enabled = false
			identityConfig.Spec.DisablePasswordAuth = false

			if err := identity.SetConfig(cmd.Context(), namespace, *identityConfig); err != nil {
				return errors.Wrap(err, "failed to set identity config")
			}

			log.FinishSpinner()

			log.ChildActionWithSpinner("Uninstalling the Identity Service")

			if err := identity.Undeploy(cmd.Context(), clientset, namespace); err != nil {
				return errors.Wrap(err, "failed to uninstall the identity service")
			}

			log.FinishSpinner()

			return nil
		},
	}

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

			log := logger.NewCLILogger(cmd.OutOrStdout())

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			log.ChildActionWithSpinner("Updating the Identity Service configuration")

			identityConfig, err := identity.GetConfig(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get identity config")
			}

			identityConfig.Spec.DisablePasswordAuth = false

			if err := identity.SetConfig(cmd.Context(), namespace, *identityConfig); err != nil {
				return errors.Wrap(err, "failed to set identity config")
			}

			log.FinishSpinner()

			return nil
		},
	}

	return cmd
}

func IdentityServiceOIDCCallbackURLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "oidc-callback-url",
		Short:         "Prints OICD callback URL",
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

			identityConfig, err := identity.GetConfig(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get identity config")
			}

			fmt.Fprintln(cmd.OutOrStdout(), identity.DexCallbackURL(identityConfig.Spec))

			return nil
		},
	}

	return cmd
}

func identityServiceDeploy(ctx context.Context, log *logger.CLILogger, clientset kubernetes.Interface, namespace string, identityConfig kotsv1beta1.IdentityConfig, ingressConfig kotsv1beta1.IngressConfig, registryConfig *kotsadmtypes.RegistryConfig, proxyEnv map[string]string, applyAppBranding bool) error {
	log.ChildActionWithSpinner("Deploying the Identity Service")

	identityConfig.Spec.Enabled = true
	identityConfig.Spec.DisablePasswordAuth = true

	if identityConfig.Spec.IngressConfig == (kotsv1beta1.IngressConfigSpec{}) {
		identityConfig.Spec.IngressConfig.Enabled = false
	} else {
		identityConfig.Spec.IngressConfig.Enabled = true
	}

	if err := identity.ValidateConfig(ctx, namespace, identityConfig, ingressConfig); err != nil {
		return errors.Wrap(err, "failed to validate identity config")
	}

	if err := identity.SetConfig(ctx, namespace, identityConfig); err != nil {
		return errors.Wrap(err, "failed to set identity config")
	}

	if err := identity.Deploy(ctx, clientset, namespace, identityConfig, ingressConfig, registryConfig, proxyEnv, applyAppBranding); err != nil {
		return errors.Wrap(err, "failed to deploy the identity service")
	}

	log.FinishSpinner()

	return nil
}

func identityServiceConfigure(ctx context.Context, log *logger.CLILogger, clientset kubernetes.Interface, namespace string, identityConfig kotsv1beta1.IdentityConfig, ingressConfig kotsv1beta1.IngressConfig, proxyEnv map[string]string, applyAppBranding bool) error {
	log.ChildActionWithSpinner("Configuring the Identity Service")

	identityConfig.Spec.Enabled = true
	identityConfig.Spec.DisablePasswordAuth = true

	if identityConfig.Spec.IngressConfig == (kotsv1beta1.IngressConfigSpec{}) {
		identityConfig.Spec.IngressConfig.Enabled = false
	} else {
		identityConfig.Spec.IngressConfig.Enabled = true
	}

	if err := identity.ValidateConfig(ctx, namespace, identityConfig, ingressConfig); err != nil {
		return errors.Wrap(err, "failed to validate identity config")
	}

	if err := identity.SetConfig(ctx, namespace, identityConfig); err != nil {
		return errors.Wrap(err, "failed to set identity config")
	}

	if err := identity.Configure(ctx, clientset, namespace, identityConfig, ingressConfig, proxyEnv, applyAppBranding); err != nil {
		return errors.Wrap(err, "failed to configure identity service")
	}

	log.FinishSpinner()

	return nil
}
