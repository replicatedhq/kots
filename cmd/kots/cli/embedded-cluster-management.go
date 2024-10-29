package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

func EmbeddedClusterConfirmManagementCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "confirm-management",
		Short:         "Confirm that the cluster is ready to deploy the application (that there are enough nodes, etc.)",
		Long:          "",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if len(args) < 1 {
				cmd.Help()
				os.Exit(1)
			}

			fmt.Print(cursor.Hide())
			defer fmt.Print(cursor.Show())

			log := logger.NewCLILogger(cmd.OutOrStdout())
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

			log.ActionWithoutSpinner("Confirming cluster management...")

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

			requestPayload := map[string]interface{}{} // empty payload
			requestBody, err := json.Marshal(requestPayload)
			if err != nil {
				return errors.Wrap(err, "failed to marshal request json")
			}

			url := fmt.Sprintf("http://localhost:%d/api/v1/embedded-cluster/management", localPort)
			newRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
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

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("unexpected status code %d from %s", resp.StatusCode, url)
			}

			log.ActionWithoutSpinner("Done")

			return nil
		},
	}
	return cmd
}
