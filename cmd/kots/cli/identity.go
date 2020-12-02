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

			log := logger.NewLogger()

			clientset, err := k8sutil.GetClientset(kubernetesConfigFlags)
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			ingressResource, err := ingress.GetConfig(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get ingress config")
			}

			identityResource := kotsv1beta1.Identity{}
			if identitySpecPath := v.GetString("identity-spec"); identitySpecPath != "" {
				content, err := ioutil.ReadFile(identitySpecPath)
				if err != nil {
					return errors.Wrap(err, "failed to read identity service spec file")
				}

				s, err := identity.DecodeSpec(content)
				if err != nil {
					return errors.Wrap(err, "failed to decoce identity service spec")
				}
				identityResource = *s
			}

			registryConfig, err := getRegistryConfig(v)
			if err != nil {
				return errors.Wrap(err, "failed to get registry config")
			}

			return identityServiceDeploy(cmd.Context(), log, clientset, namespace, identityResource, *ingressResource, registryConfig)
		},
	}

	cmd.Flags().String("identity-spec", "", "path to a kots.Identity resource file")

	// random other registry flags
	cmd.Flags().String("license-file", "", "path to a license file to use when download a replicated app")
	cmd.Flags().String("airgap-bundle", "", "path to the application airgap bundle where application metadata will be loaded from")
	cmd.Flags().Bool("airgap", false, "set to true to run install in airgapped mode. setting --airgap-bundle implies --airgap=true.")

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

			log := logger.NewLogger()

			clientset, err := k8sutil.GetClientset(kubernetesConfigFlags)
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			ingressResource, err := ingress.GetConfig(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get ingress config")
			}

			identityResource := kotsv1beta1.Identity{}
			if identitySpecPath := v.GetString("identity-spec"); identitySpecPath != "" {
				content, err := ioutil.ReadFile(identitySpecPath)
				if err != nil {
					return errors.Wrap(err, "failed to read identity service spec file")
				}

				s, err := identity.DecodeSpec(content)
				if err != nil {
					return errors.Wrap(err, "failed to decoce identity service spec")
				}
				identityResource = *s
			}

			return identityServiceConfigure(cmd.Context(), log, clientset, namespace, identityResource, *ingressResource)
		},
	}

	cmd.Flags().String("identity-config", "", "path to a yaml file containing the KOTS identity service configuration")

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

			log := logger.NewLogger()

			clientset, err := k8sutil.GetClientset(kubernetesConfigFlags)
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			log.ChildActionWithSpinner("Updating the Identity Service configuration")

			identityResource, err := identity.GetConfig(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get identity config")
			}

			identityResource.Spec.Enabled = false
			identityResource.Spec.DisablePasswordAuth = false

			if err := identity.SetConfig(cmd.Context(), namespace, *identityResource); err != nil {
				return errors.Wrap(err, "failed to set identity config")
			}

			log.FinishSpinner()

			log.ChildActionWithSpinner("Uninstalling the Identity Service")

			if err := identity.Undeploy(cmd.Context(), log, clientset, namespace); err != nil {
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

			log := logger.NewLogger()

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			log.ChildActionWithSpinner("Updating the Identity Service configuration")

			identityResource, err := identity.GetConfig(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get identity config")
			}

			identityResource.Spec.DisablePasswordAuth = false

			if err := identity.SetConfig(cmd.Context(), namespace, *identityResource); err != nil {
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

			identityResource, err := identity.GetConfig(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get identity config")
			}

			fmt.Fprintln(cmd.OutOrStdout(), identity.DexCallbackURL(identityResource.Spec))

			return nil
		},
	}

	return cmd
}

func identityServiceDeploy(ctx context.Context, log *logger.Logger, clientset kubernetes.Interface, namespace string, identityResource kotsv1beta1.Identity, ingressResource kotsv1beta1.Ingress, registryConfig *kotsadmtypes.KotsadmOptions) error {
	log.ChildActionWithSpinner("Deploying the Identity Service")

	identityResource.Spec.Enabled = true
	identityResource.Spec.DisablePasswordAuth = true

	if identityResource.Spec.IngressConfig == (kotsv1beta1.IngressSpec{}) {
		identityResource.Spec.IngressConfig.Enabled = false
	} else {
		identityResource.Spec.IngressConfig.Enabled = true
	}

	if err := identity.ConfigValidate(identityResource.Spec, ingressResource.Spec); err != nil {
		return errors.Wrap(err, "failed to validate identity config")
	}

	if err := identity.SetConfig(ctx, namespace, identityResource); err != nil {
		return errors.Wrap(err, "failed to set identity config")
	}

	if err := identity.Deploy(ctx, clientset, namespace, identityResource, ingressResource, registryConfig); err != nil {
		return errors.Wrap(err, "failed to deploy the identity service")
	}

	log.FinishSpinner()

	return nil
}

func identityServiceConfigure(ctx context.Context, log *logger.Logger, clientset kubernetes.Interface, namespace string, identityResource kotsv1beta1.Identity, ingressResource kotsv1beta1.Ingress) error {
	log.ChildActionWithSpinner("Configuring the Identity Service")

	identityResource.Spec.Enabled = true
	identityResource.Spec.DisablePasswordAuth = true

	if identityResource.Spec.IngressConfig == (kotsv1beta1.IngressSpec{}) {
		identityResource.Spec.IngressConfig.Enabled = false
	} else {
		identityResource.Spec.IngressConfig.Enabled = true
	}

	if err := identity.ConfigValidate(identityResource.Spec, ingressResource.Spec); err != nil {
		return errors.Wrap(err, "failed to validate identity config")
	}

	if err := identity.SetConfig(ctx, namespace, identityResource); err != nil {
		return errors.Wrap(err, "failed to set identity config")
	}

	if err := identity.Configure(ctx, clientset, namespace, identityResource, ingressResource); err != nil {
		return errors.Wrap(err, "failed to configure identity service")
	}

	log.FinishSpinner()

	return nil
}
