package cli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

			log := logger.NewCLILogger()

			// use namespace-as-arg if provided, else use namespace from -n/--namespace
			namespace := v.GetString("namespace")
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

			podName, err := k8sutil.FindKotsadm(clientset, namespace)
			if err != nil {
				return errors.Wrap(err, "failed to find kotsadm pod")
			}

			localPort, errChan, err := k8sutil.PortForward(0, 3000, namespace, podName, false, stopCh, log)
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

			newReq, err := http.NewRequest("POST", url, nil)
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

			b, err := ioutil.ReadAll(resp.Body)
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

	return cmd
}
