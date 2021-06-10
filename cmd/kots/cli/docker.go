package cli

import (
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"

	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func DockerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker",
		Short: "KOTS Docker interface",
	}

	cmd.AddCommand(DockerEnsureSecretCmd())

	return cmd
}

func DockerEnsureSecretCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ensure-secret",
		Short:         "Creates an image pull secret that the Admin Console can utilize in case of rate limiting.",
		Long:          `Will validate the credentials before creating the image pull secret`,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			dockerHubUsername := v.GetString("dockerhub-username")
			if dockerHubUsername == "" {
				return errors.New("--dockerhub-username flag is required")
			}

			dockerHubPassword := v.GetString("dockerhub-password")
			if dockerHubPassword == "" {
				return errors.New("--dockerhub-password flag is required")
			}

			// validate credentials
			sysCtx := &types.SystemContext{DockerDisableV1Ping: true}
			if err := docker.CheckAuth(cmd.Context(), sysCtx, dockerHubUsername, dockerHubPassword, registry.DockerHubRegistryName); err != nil {
				return errors.Wrap(err, "failed to authenticate to docker")
			}

			// create the image pull secret
			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			if err := registry.EnsureDockerHubSecret(dockerHubUsername, dockerHubPassword, namespace, clientset); err != nil {
				return errors.Wrap(err, "failed to ensure dockerhub secret")
			}

			return nil
		},
	}

	cmd.Flags().String("dockerhub-username", "", "DockerHub username to be used")
	cmd.Flags().String("dockerhub-password", "", "DockerHub password to be used")

	return cmd
}
