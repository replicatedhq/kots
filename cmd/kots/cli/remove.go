package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	cursor "github.com/ahmetalpbalkan/go-cursor"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "remove [slug]",
		Short:         "Remove an application from console",
		Long:          `Remove application reference identified by slug from Admin Console.  This command does not remove application resources from the cluster.`,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if len(args) != 1 {
				cmd.Help()
				os.Exit(1)
			}

			fmt.Print(cursor.Hide())
			defer fmt.Print(cursor.Show())

			log := logger.NewCLILogger(cmd.OutOrStdout())
			appSlug := args[0]
			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			getPodName := func() (string, error) {
				return k8sutil.WaitForKotsadm(clientset, namespace, time.Second*5)
			}

			stopCh := make(chan struct{})
			defer close(stopCh)

			if v.GetBool("undeploy") {
				log.ActionWithSpinner("Removing application %s reference from Admin Console and deleting associated resources from the cluster", appSlug)
			} else {
				log.ActionWithSpinner("Removing application %s reference from Admin Console", appSlug)
			}

			localPort, errChan, err := k8sutil.PortForward(0, 3000, namespace, getPodName, false, stopCh, log)
			if err != nil {
				log.FinishSpinnerWithError()
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
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to get kotsadm auth slug")
			}

			requestPayload := map[string]interface{}{
				"undeploy": v.GetBool("undeploy"),
				"force":    v.GetBool("force"),
			}

			requestBody, err := json.Marshal(requestPayload)
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to marshal request json")
			}

			url := fmt.Sprintf("http://localhost:%d/api/v1/app/%s/remove", localPort, url.QueryEscape(appSlug))
			newRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to create http request")
			}
			newRequest.Header.Add("Authorization", authSlug)
			newRequest.Header.Add("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(newRequest)
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to execute http request")
			}
			defer resp.Body.Close()

			b, err := io.ReadAll(resp.Body)
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to read server response")
			}

			type removeAppResponse struct {
				Error string `json:"error"`
			}
			response := removeAppResponse{}
			_ = json.Unmarshal(b, &response)

			if resp.StatusCode != http.StatusOK {
				if resp.StatusCode == http.StatusNotFound {
					log.FinishSpinnerWithError()
					return errors.Errorf("app with slug %s not found", appSlug)
				} else if resp.StatusCode == http.StatusBadRequest {
					if v.GetBool("force") {
						log.FinishSpinnerWithError()
						return errors.Wrap(errors.New(response.Error), "failed to remove app")
					} else {
						log.FinishSpinnerWithError()
						return errors.Errorf("Application is already deployed. Re-run the command with --force flag to remove application reference anyway.")
					}
				} else {
					log.FinishSpinnerWithError()
					return errors.Wrapf(errors.New(response.Error), "unexpected status code from %v", resp.StatusCode)
				}
			}

			log.FinishSpinner()
			log.ActionWithoutSpinner("Application %s has been removed", appSlug)

			return nil
		},
	}

	cmd.Flags().StringP("namespace", "n", "", "the namespace in which kots/kotsadm is installed")
	cmd.Flags().Bool("undeploy", false, "undeploy/delete application resources from the cluster")
	cmd.Flags().BoolP("force", "f", false, "removing application reference even if it was already deployed")

	return cmd
}
