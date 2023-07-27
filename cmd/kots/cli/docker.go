package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	cursor "github.com/ahmetalpbalkan/go-cursor"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"

	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
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

			log := logger.NewCLILogger(cmd.OutOrStdout())

			// validate credentials
			sysCtx := &types.SystemContext{DockerDisableV1Ping: true}
			if err := docker.CheckAuth(cmd.Context(), sysCtx, dockerHubUsername, dockerHubPassword, registry.DockerHubRegistryName); err != nil {
				return errors.Wrap(err, "failed to authenticate to docker")
			}

			// create the image pull secret
			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			err = registry.EnsureDockerHubSecret(dockerHubUsername, dockerHubPassword, namespace, clientset)
			if err != nil {
				if err == registry.ErrDockerHubCredentialsExist {
					log.Info("New application version will not be created because secret %q with the same credentials already exists.", registry.DockerHubSecretName)
					return nil
				}

				return errors.Wrap(err, "failed to ensure dockerhub secret")
			}

			fmt.Print(cursor.Hide())
			defer fmt.Print(cursor.Show())

			getPodName := func() (string, error) {
				return k8sutil.WaitForKotsadm(clientset, namespace, time.Second*5)
			}

			stopCh := make(chan struct{})
			defer close(stopCh)

			log.ActionWithSpinner("Updating applications")
			defer log.FinishSpinnerWithError()

			localPort, errChan, err := k8sutil.PortForward(0, 3000, namespace, getPodName, false, stopCh, log)
			if err != nil {
				return errors.Wrap(err, "failed to start port forwarding")
			}

			go func() {
				select {
				case err := <-errChan:
					if err != nil {
						log.Error(err)
					}
				case <-stopCh:
				}
			}()

			authSlug, err := auth.GetOrCreateAuthSlug(clientset, namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get kotsadm auth slug")
			}

			url := fmt.Sprintf("http://localhost:%d/api/v1/docker/secret-updated", localPort)
			newRequest, err := http.NewRequest("POST", url, nil)
			if err != nil {
				return errors.Wrap(err, "failed to create http request")
			}
			newRequest.Header.Add("Authorization", authSlug)
			newRequest.Header.Add("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(newRequest)
			if err != nil {
				return errors.Wrap(err, "failed to execute http request")
			}
			defer resp.Body.Close()

			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return errors.Wrap(err, "failed to read server response")
			}

			response := struct {
				Success bool   `json:"success"`
				Error   string `json:"error"`
			}{}
			_ = json.Unmarshal(b, &response)

			if resp.StatusCode != http.StatusOK {
				return errors.Wrapf(errors.New(response.Error), "unexpected status code from %v", resp.StatusCode)
			}

			log.FinishSpinner()

			return nil
		},
	}

	cmd.Flags().String("dockerhub-username", "", "DockerHub username to be used")
	cmd.Flags().String("dockerhub-password", "", "DockerHub password to be used")

	return cmd
}
