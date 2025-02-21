package cli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upload"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func UpstreamDownloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "download [appSlug]",
		Short:         "Retry downloading a failed update of the upstream application",
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
			appSlug := args[0]

			appSequence := v.GetInt64("sequence")
			if appSequence == -1 {
				return errors.New("--sequence flag is required")
			}

			output := v.GetString("output")
			if output != "json" && output != "" {
				return errors.Errorf("output format %s not supported (allowed formats are: json)", output)
			}

			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}

			stopCh := make(chan struct{})
			defer close(stopCh)

			log := logger.NewCLILogger(cmd.OutOrStdout())
			if output != "" {
				log.Silence()
			}

			localPort, errChan, err := upload.StartPortForward(namespace, stopCh, log)
			if err != nil {
				return err
			}

			urlVals := url.Values{}
			if v.GetBool("skip-preflights") {
				urlVals.Set("skipPreflights", "true")
			}
			if v.GetBool("skip-compatibility-check") {
				urlVals.Set("skipCompatibilityCheck", "true")
			}
			if v.GetBool("wait") {
				urlVals.Set("wait", "true")
			}
			url := fmt.Sprintf("http://localhost:%d/api/v1/app/%s/sequence/%d/download?%s", localPort, url.PathEscape(appSlug), appSequence, urlVals.Encode())

			go func() {
				select {
				case err := <-errChan:
					if err != nil {
						log.Error(err)
						os.Exit(-1)
					}
				case <-stopCh:
				}
			}()

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get k8s clientset")
			}

			authSlug, err := auth.GetOrCreateAuthSlug(clientset, namespace)
			if err != nil {
				log.Info("Unable to authenticate to the Admin Console running in the %s namespace. Ensure you have read access to secrets in this namespace and try again.", namespace)
				if v.GetBool("debug") {
					return errors.Wrap(err, "failed to get kotsadm auth slug")
				}
				os.Exit(2) // not returning error here as we don't want to show the entire stack trace to normal users
			}

			log.ActionWithSpinner("Retrying download for sequence %d", appSequence)

			newReq, err := http.NewRequest("POST", url, nil)
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to create update check request")
			}
			newReq.Header.Add("Authorization", authSlug)
			resp, err := http.DefaultClient.Do(newReq)
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to check for updates")
			}
			defer resp.Body.Close()

			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to read server response")
			}

			type Response struct {
				Success bool   `json:"success"`
				Error   string `json:"error,omitempty"`
			}
			r := Response{}
			if err := json.Unmarshal(b, &r); err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to parse response")
			}

			if output == "json" {
				outputJSON, err := json.Marshal(r)
				if err != nil {
					log.FinishSpinnerWithError()
					return errors.Wrap(err, "error marshaling JSON")
				}
				log.FinishSpinner()
				fmt.Println(string(outputJSON))
				return nil
			}

			if r.Error != "" {
				log.FinishSpinnerWithError()
				log.Error(errors.New(r.Error))
				return errors.Errorf("Unexpected response from the API: %d", resp.StatusCode)
			}

			log.FinishSpinner()

			if v.GetBool("wait") {
				log.ActionWithoutSpinner("Downloaded successfully.")
			} else {
				log.ActionWithoutSpinner("App sequence %d is being re-downloaded.", appSequence)
			}

			return nil
		},
	}

	cmd.Flags().Int64("sequence", -1, "local app sequence for the version to retry downloading.")
	cmd.Flags().Bool("skip-preflights", false, "set to true to skip preflight checks")
	cmd.Flags().Bool("skip-compatibility-check", false, "set to true to skip compatibility checks between the current kots version and the update")
	cmd.Flags().Bool("wait", true, "set to false to download the update in the background")
	cmd.Flags().StringP("output", "o", "", "output format (currently supported: json)")

	cmd.Flags().Bool("debug", false, "when set, log full error traces in some cases where we provide a pretty message")
	cmd.Flags().MarkHidden("debug")

	return cmd
}
