package cli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func UpstreamUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "upgrade [appSlug]",
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

			clientset, err := k8sutil.GetClientset(kubernetesConfigFlags)
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to get clientset")
			}

			podName, err := k8sutil.FindKotsadm(clientset, v.GetString("namespace"))
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to find kotsadm pod")
			}

			localPort, errChan, err := k8sutil.PortForward(kubernetesConfigFlags, 0, 3000, v.GetString("namespace"), podName, false, stopCh, log)
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
			updateCheckURI := fmt.Sprintf("http://localhost:%d/api/v1/app/%s/updatecheck", localPort, appSlug)
			if viper.GetBool("deploy") {
				updateCheckURI = fmt.Sprintf("%s?deploy=true", updateCheckURI)
			}

			authSlug, err := auth.GetOrCreateAuthSlug(kubernetesConfigFlags, v.GetString("namespace"))
			if err != nil {
				log.FinishSpinnerWithError()
				log.Info("Unable to authenticate to the Admin Console running in the %s namespace. Ensure you have read access to secrets in this namespace and try again.", v.GetString("namespace"))
				if v.GetBool("debug") {
					return errors.Wrap(err, "failed to get kotsadm auth slug")
				}
				os.Exit(2) // not returning error here as we don't want to show the entire stack trace to normal users
			}

			newReq, err := http.NewRequest("POST", updateCheckURI, strings.NewReader("{}"))
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to create update check request")
			}
			newReq.Header.Add("Content-Type", "application/json")
			newReq.Header.Add("Authorization", authSlug)
			resp, err := http.DefaultClient.Do(newReq)
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to check for updates")
			}
			defer resp.Body.Close()

			if resp.StatusCode == 404 {
				log.FinishSpinnerWithError()
				return errors.Errorf("The application %s was not found in the cluster in the specified namespace", args[0])
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
				AvailableUpdates int `json:"availableUpdates"`
			}
			ucr := updateCheckResponse{}
			if err := json.Unmarshal(b, &ucr); err != nil {
				return errors.Wrap(err, "failed to parse response")
			}

			log.FinishSpinner()

			if viper.GetBool("deploy") {
				if ucr.AvailableUpdates == 0 {
					log.ActionWithoutSpinner("")
					log.ActionWithoutSpinner("There are no application updates available, ensuring latest is marked as deployed")
				} else {
					log.ActionWithoutSpinner("")
					log.ActionWithoutSpinner(fmt.Sprintf("There are currently %d updates available in the Admin Console, when the latest release is downloaded, it will be deployed", ucr.AvailableUpdates))
				}

				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner("To access the Admin Console, run kubectl kots admin-console --namespace %s", v.GetString("namespace"))
				log.ActionWithoutSpinner("")

				return nil
			}

			if ucr.AvailableUpdates == 0 {
				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner("There are no application updates available")
			} else {
				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner(fmt.Sprintf("There are currently %d updates available in the Admin Console", ucr.AvailableUpdates))
			}

			log.ActionWithoutSpinner("To access the Admin Console, run kubectl kots admin-console --namespace %s", v.GetString("namespace"))
			log.ActionWithoutSpinner("")

			return nil
		},
	}

	cmd.Flags().Bool("deploy", false, "when set, automatically deploy the latest version")

	cmd.Flags().Bool("debug", false, "when set, log full error traces in some cases where we provide a pretty message")
	cmd.Flags().MarkHidden("debug")

	return cmd
}
