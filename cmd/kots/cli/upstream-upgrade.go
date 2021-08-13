package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upload"
	"github.com/replicatedhq/kots/pkg/upstream"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type UpstreamUpgradeOutput struct {
	Success          bool   `json:"success"`
	AvailableUpdates int64  `json:"availableUpdates,omitempty"`
	Error            string `json:"error,omitempty"`
}

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

			appSlug := args[0]

			isKurl, err := kotsadm.IsKurl()
			if err != nil {
				return errors.Wrap(err, "failed to check kURL")
			}

			output := v.GetString("output")
			if output != "json" && output != "" {
				return errors.Errorf("output format %s not supported (allowed formats are: json)", output)
			}

			upgradeOptions := upstream.UpgradeOptions{
				AirgapBundle:      v.GetString("airgap-bundle"),
				RegistryEndpoint:  v.GetString("kotsadm-namespace"),
				RegistryNamespace: v.GetString("registry-namespace"),
				RegistryUsername:  v.GetString("registry-username"),
				RegistryPassword:  v.GetString("registry-password"),
				IsKurl:            isKurl,
				DisableImagePush:  v.GetBool("disable-image-push"),
				Namespace:         v.GetString("namespace"),
				Debug:             v.GetBool("debug"),
				Deploy:            v.GetBool("deploy"),
				Silent:            output != "",
			}

			stopCh := make(chan struct{})
			defer close(stopCh)

			log := logger.NewCLILogger()
			localPort, errChan, err := upload.StartPortForward(v.GetString("namespace"), stopCh, log)
			if err != nil {
				return err
			}

			urlVals := url.Values{}
			if viper.GetBool("deploy") {
				urlVals.Set("deploy", "true")
			}
			if viper.GetBool("skip-preflights") {
				urlVals.Set("skipPreflights", "true")
			}
			if viper.GetBool("is-cli") {
				urlVals.Set("isCLI", "true")
			}
			upgradeOptions.UpdateCheckEndpoint = fmt.Sprintf("http://localhost:%d/api/v1/app/%s/updatecheck?%s", localPort, url.PathEscape(appSlug), urlVals.Encode())

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

			var upgradeOutput UpstreamUpgradeOutput
			res, err := upstream.Upgrade(appSlug, upgradeOptions)
			if err != nil && output == "" {
				return err
			} else if err != nil {
				upgradeOutput.Error = fmt.Sprint(err)
			} else {
				upgradeOutput.Success = true
				upgradeOutput.AvailableUpdates = res.AvailableUpdates
			}

			if output == "json" {
				outputJSON, err := json.Marshal(upgradeOutput)
				if err != nil {
					return errors.Wrap(err, "error marshaling JSON")
				}
				log.Info(string(outputJSON))
			}

			return nil
		},
	}

	cmd.Flags().Bool("deploy", false, "when set, automatically deploy the latest version")
	cmd.Flags().Bool("skip-preflights", false, "set to true to skip preflight checks")

	cmd.Flags().String("airgap-bundle", "", "path to the application airgap bundle where application images and metadata will be loaded from")
	cmd.Flags().String("kotsadm-registry", "", "registry endpoint where application images will be pushed")
	cmd.Flags().String("kotsadm-namespace", "", "registry namespace to use for application images")
	cmd.Flags().String("registry-username", "", "user name to use to authenticate with the registry")
	cmd.Flags().String("registry-password", "", "password to use to authenticate with the registry")
	cmd.Flags().Bool("disable-image-push", false, "set to true to disable images from being pushed to private registry")
	cmd.Flags().StringP("output", "o", "", "output format (currently supported: json)")

	cmd.Flags().Bool("debug", false, "when set, log full error traces in some cases where we provide a pretty message")
	cmd.Flags().MarkHidden("debug")

	return cmd
}
