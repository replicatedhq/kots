package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func GarbageCollectImagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "garbage-collect-images [namespace]",
		Short:         "Run image garbage collection",
		Long:          `Triggers image garbage collection for all apps`,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			log := logger.NewCLILogger(cmd.OutOrStdout())

			// use namespace-as-arg if provided, else use namespace from -n/--namespace
			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}
			if len(args) == 1 {
				namespace = args[0]
			} else if len(args) > 1 {
				fmt.Printf("more than one argument supplied: %+v\n", args)
				os.Exit(1)
			}

			if err := validateNamespace(namespace); err != nil {
				return errors.Wrap(err, "failed to validate namespace")
			}

			stopCh := make(chan struct{})
			defer close(stopCh)

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			getPodName := func() (string, error) {
				return k8sutil.FindKotsadm(clientset, namespace)
			}

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

			url := fmt.Sprintf("http://localhost:%d/api/v1/garbage-collect-images", localPort)

			authSlug, err := auth.GetOrCreateAuthSlug(clientset, namespace)
			if err != nil {
				log.Info("Unable to authenticate to the Admin Console running in the %s namespace. Ensure you have read access to secrets in this namespace and try again.", namespace)
				if v.GetBool("debug") {
					return errors.Wrap(err, "failed to get kotsadm auth slug")
				}
				os.Exit(2) // not returning error here as we don't want to show the entire stack trace to normal users
			}

			requestPayload := map[string]interface{}{
				"ignoreRollback": v.GetBool("ignore-rollback"),
			}
			requestBody, err := json.Marshal(requestPayload)
			if err != nil {
				return errors.Wrap(err, "failed to marshal request json")
			}
			newReq, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
			if err != nil {
				return errors.Wrap(err, "failed to create request")
			}
			newReq.Header.Add("Content-Type", "application/json")
			newReq.Header.Add("Authorization", authSlug)

			resp, err := http.DefaultClient.Do(newReq)
			if err != nil {
				return errors.Wrap(err, "failed to check for updates")
			}
			defer resp.Body.Close()

			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return errors.Wrap(err, "failed to read")
			}

			type Response struct {
				Error string `json:"error"`
			}
			response := Response{}
			if err = json.Unmarshal(b, &response); err != nil {
				return errors.Wrapf(err, "failed to unmarshal server response: %s", b)
			}

			if response.Error != "" {
				return errors.New(response.Error)
			}

			if resp.StatusCode != http.StatusOK {
				return errors.Errorf("unexpected response from server %v: %s", resp.StatusCode, b)
			}

			log.ActionWithoutSpinner("Garbage collection has been triggered")

			return nil
		},
	}

	cmd.Flags().Bool("ignore-rollback", false, "force images garbage collection even if rollback is enabled for the application")

	return cmd
}
