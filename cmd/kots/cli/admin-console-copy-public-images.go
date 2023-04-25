package cli

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	dockerregistry "github.com/replicatedhq/kots/pkg/docker/registry"
	registrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
)

func AdminCopyPublicImagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "copy-public-images [registry host]",
		Short:         "Copy public admin console images",
		Long:          "Copy public admin console images to a private registry",
		Hidden:        true,
		SilenceUsage:  true,
		SilenceErrors: false,
		Args:          cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()
			log := logger.NewCLILogger(cmd.OutOrStdout())

			endpoint := args[0]
			hostname, err := getHostnameFromEndpoint(endpoint)
			if err != nil {
				return errors.Wrap(err, "failed get hostname from endpoint")
			}

			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}

			username := v.GetString("registry-username")
			password := v.GetString("registry-password")
			if username == "" && password == "" {
				u, p, err := getRegistryCredentialsFromSecret(hostname, namespace)
				if err != nil {
					if !kuberneteserrors.IsNotFound(err) {
						log.Info("Failed to find registry credentials, will try to push anonymously: %v", err)
					}
				} else {
					username, password = u, p
				}
			}

			if registry.IsECREndpoint(endpoint) && username != "AWS" {
				var err error
				login, err := registry.GetECRLogin(endpoint, username, password)
				if err != nil {
					return errors.Wrap(err, "failed get ecr login")
				}
				username = login.Username
				password = login.Password
			}

			if !v.GetBool("skip-registry-check") {
				log.ActionWithSpinner("Validating registry information")

				if err := dockerregistry.CheckAccess(hostname, username, password); err != nil {
					log.FinishSpinnerWithError()
					return fmt.Errorf("Failed to test access to %q with user %q: %v", hostname, username, err)
				}
				log.FinishSpinner()
			}

			options := kotsadmtypes.PushImagesOptions{
				KotsadmTag: v.GetString("kotsadm-tag"),
				Registry: registrytypes.RegistryOptions{
					Endpoint: endpoint,
					Username: username,
					Password: password,
				},
				ProgressWriter: os.Stdout,
			}

			err = kotsadm.CopyImages(options, namespace)
			if err != nil {
				return errors.Wrap(err, "failed to copy images")
			}

			return nil
		},
	}

	cmd.Flags().String("registry-username", "", "user name to use to authenticate with the registry")
	cmd.Flags().String("registry-password", "", "password to use to authenticate with the registry")
	cmd.Flags().Bool("skip-registry-check", false, "skip the connectivity test and validation of the provided registry information")

	cmd.Flags().String("source-registry", "docker.io", "source registry for public images")

	cmd.Flags().String("kotsadm-tag", "", "set to override the tag of kotsadm. this may create an incompatible deployment because the version of kots and kotsadm are designed to work together")
	cmd.Flags().MarkHidden("kotsadm-tag")

	return cmd
}
