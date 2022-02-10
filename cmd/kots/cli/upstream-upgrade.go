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
				AirgapBundle:       v.GetString("airgap-bundle"),
				RegistryEndpoint:   v.GetString("kotsadm-registry"),
				RegistryNamespace:  v.GetString("kotsadm-namespace"),
				RegistryUsername:   v.GetString("registry-username"),
				RegistryPassword:   v.GetString("registry-password"),
				IsKurl:             isKurl,
				DisableImagePush:   v.GetBool("disable-image-push"),
				Namespace:          v.GetString("namespace"),
				Debug:              v.GetBool("debug"),
				Deploy:             v.GetBool("deploy"),
				DeployVersionLabel: v.GetString("deploy-version-label"),
				Wait:               v.GetBool("wait"),
				Silent:             output != "",
			}

			stopCh := make(chan struct{})
			defer close(stopCh)

			log := logger.NewCLILogger()
			localPort, errChan, err := upload.StartPortForward(v.GetString("namespace"), stopCh, log)
			if err != nil {
				return err
			}

			urlVals := url.Values{}
			if v.GetBool("deploy") {
				urlVals.Set("deploy", "true")
			}
			if dvl := v.GetString("deploy-version-label"); dvl != "" {
				urlVals.Set("deployVersionLabel", dvl)
			}
			if v.GetBool("skip-preflights") {
				urlVals.Set("skipPreflights", "true")
			}
			if v.GetBool("skip-compatibility-check") {
				urlVals.Set("skipCompatibilityCheck", "true")
			}
			if v.GetBool("is-cli") {
				urlVals.Set("isCLI", "true")
			}
			if v.GetBool("wait") {
				urlVals.Set("wait", "true")
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

			res, err := upstream.Upgrade(appSlug, upgradeOptions)
			if err != nil {
				res = &upstream.UpgradeResponse{
					Error: fmt.Sprint(err),
				}
			} else {
				res.Success = true
			}

			err = logUpstreamUpgrade(log, res, output)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().Bool("deploy", false, "when set, automatically deploy the latest version. if an airgap bundle is provided, the version created from that airgap bundle is deployed instead.")
	cmd.Flags().String("deploy-version-label", "", "when set, automatically deploy the version with the provided version label")
	cmd.Flags().Bool("skip-preflights", false, "set to true to skip preflight checks")
	cmd.Flags().Bool("skip-compatibility-check", false, "set to true to skip compatibility checks between the current kots version and new app version(s)")
	cmd.Flags().Bool("wait", true, "set to false to download the updates in the background")

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

func logUpstreamUpgrade(log *logger.CLILogger, res *upstream.UpgradeResponse, output string) error {
	if output == "json" {
		outputJSON, err := json.Marshal(res)
		if err != nil {
			return errors.Wrap(err, "error marshaling JSON")
		}
		log.Info(string(outputJSON))
		return nil
	}

	// text output
	if res.Error != "" {
		log.ActionWithoutSpinner(res.Error)
	} else {
		if res.CurrentRelease != nil {
			log.ActionWithoutSpinner(fmt.Sprintf("Currently deployed release: sequence %v, version %v", res.CurrentRelease.Sequence, res.CurrentRelease.Version))
		}

		for _, r := range res.AvailableReleases {
			log.ActionWithoutSpinner(fmt.Sprintf("Downloading available release: sequence %v, version %v", r.Sequence, r.Version))
		}

		if res.DeployingRelease != nil {
			log.ActionWithoutSpinner(fmt.Sprintf("Deploying release: sequence %v, version %v", res.DeployingRelease.Sequence, res.DeployingRelease.Version))
		}
	}

	return nil
}
