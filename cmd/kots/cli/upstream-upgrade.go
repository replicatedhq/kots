package cli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func UpstreamUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "upgrade",
		Short:         "Fetch the latest version of the upstream application",
		Long:          "",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if len(args) == 0 {
				cmd.Help()
				os.Exit(1)
			}

			log := logger.NewLogger()
			log.ActionWithSpinner("Checking for application updates")

			stopCh := make(chan struct{})
			defer close(stopCh)

			podName, err := k8sutil.FindKotsadm(v.GetString("namespace"))
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to find kotsadm pod")
			}

			errChan, err := k8sutil.PortForward(v.GetString("kubeconfig"), 3000, 3000, v.GetString("namespace"), podName, false, stopCh, log)
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

			appSlug := args[0]
			resp, err := http.Post(fmt.Sprintf("http://localhost:3000/api/v1/kots/%s/update-check", appSlug), "application/json", strings.NewReader("{}"))
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to check for updates")
			}
			defer resp.Body.Close()

			if resp.StatusCode == 404 {
				log.FinishSpinnerWithError()
				return errors.New("The application was not found in the cluster in the specified namespace")
			} else if resp.StatusCode != 200 {
				log.FinishSpinnerWithError()
				return errors.Errorf("Unexpected response from the API: %d", resp.StatusCode)
			}

			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to read server response")
			}

			type updateCheckResponse struct {
				UpdatesAvailable int `json:"updatesAvailable"`
			}
			ucr := updateCheckResponse{}
			if err := json.Unmarshal(b, &ucr); err != nil {
				return errors.Wrap(err, "failed to parse response")
			}

			if ucr.UpdatesAvailable == 0 {
				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner("There are no application updates available")
				log.ActionWithoutSpinner("")
			} else {
				if !viper.GetBool("deploy") {
					log.ActionWithoutSpinner("")
					log.ActionWithoutSpinner(fmt.Sprintf("There are currently %d updates available in the Admin Console", ucr.UpdatesAvailable))
					log.ActionWithoutSpinner("To access the Admin Console, run kubectl kots admin-console --namespace %s", v.GetString("namespace"))
					log.ActionWithoutSpinner("")
				}

				// Apply the latest version
				_, err := http.Post(fmt.Sprintf("http://localhost:3000/api/v1/kots/%s/deploy-latest", appSlug), "application/json", strings.NewReader("{}"))
				if err != nil {
					log.FinishSpinnerWithError()
					return errors.Wrap(err, "failed to deploy latest")
				}
			}

			return nil
		},
	}

	cmd.Flags().String("kubeconfig", defaultKubeConfig(), "the kubeconfig to use")
	cmd.Flags().StringP("namespace", "n", "default", "the namespace where the admin console is running")
	cmd.Flags().Bool("deploy", false, "when set, automatically deploy the latest version downloads")

	return cmd
}
