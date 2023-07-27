package cli

import (
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

func AppStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "app-status [appSlug]",
		Short:         "Returns the app status",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		Hidden:        true,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			appSlug := v.GetString("slug")
			// similar to how "download" works, we support flags and args?
			if appSlug == "" {
				if len(args) == 1 {
					appSlug = args[0]
				} else {
					cmd.Help()
					os.Exit(1)
				}
			}

			log := logger.NewCLILogger(cmd.OutOrStdout())

			stopCh := make(chan struct{})
			defer close(stopCh)

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}

			getPodName := func() (string, error) {
				return k8sutil.FindKotsadm(clientset, namespace)
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

			url := fmt.Sprintf("http://localhost:%d/api/v1/app/%s/status", localPort, appSlug)

			authSlug, err := auth.GetOrCreateAuthSlug(clientset, v.GetString("namespace"))
			if err != nil {
				log.FinishSpinnerWithError()
				log.Info("Unable to authenticate to the Admin Console running in the %s namespace. Ensure you have read access to secrets in this namespace and try again.", v.GetString("namespace"))
				if v.GetBool("debug") {
					return errors.Wrap(err, "failed to get kotsadm auth slug")
				}
				os.Exit(2) // not returning error here as we don't want to show the entire stack trace to normal users
			}

			newReq, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return errors.Wrap(err, "failed to create request")
			}
			newReq.Header.Add("Content-Type", "application/json")
			newReq.Header.Add("Authorization", authSlug)

			resp, err := http.DefaultClient.Do(newReq)
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to check for updates")
			}
			defer resp.Body.Close()

			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return errors.Wrap(err, "failed to read")
			}

			fmt.Printf("%s\n", b)

			return nil
		},
	}

	cmd.Flags().StringP("namespace", "n", "default", "namespace in which kots/kotsadm is installed")
	cmd.Flags().String("slug", "", "the application slug to get the status of")

	return cmd
}
