package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/pkg/errors"
	dockerregistry "github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kurl"
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
			log := logger.NewCLILogger(cmd.OutOrStdout())

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get k8s clientset")
			}

			isKurl, err := kurl.IsKurl(clientset)
			if err != nil {
				return errors.Wrap(err, "failed to check if cluster is kurl")
			}

			output := v.GetString("output")
			if output != "json" && output != "" {
				return errors.Errorf("output format %s not supported (allowed formats are: json)", output)
			}

			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}

			registryConfig, err := getRegistryConfig(v, clientset, appSlug)
			if err != nil {
				return errors.Wrap(err, "failed to get registry config")
			}

			if registryConfig.OverrideRegistry != "" && !v.GetBool("skip-registry-check") {
				log.ActionWithSpinner("Validating registry information")

				host, err := getHostFromEndpoint(registryConfig.OverrideRegistry)
				if err != nil {
					log.FinishSpinnerWithError()
					return errors.Wrap(err, "failed get host from endpoint")
				}

				if err := dockerregistry.CheckAccess(host, registryConfig.Username, registryConfig.Password); err != nil {
					log.FinishSpinnerWithError()
					return fmt.Errorf("Failed to test access to %q with user %q: %v", host, registryConfig.Username, err)
				}

				log.FinishSpinner()
			}

			upgradeOptions := upstream.UpgradeOptions{
				AirgapBundle:       v.GetString("airgap-bundle"),
				RegistryConfig:     *registryConfig,
				IsKurl:             isKurl,
				DisableImagePush:   v.GetBool("disable-image-push"),
				Namespace:          namespace,
				Debug:              v.GetBool("debug"),
				Deploy:             v.GetBool("deploy"),
				DeployVersionLabel: v.GetString("deploy-version-label"),
				Wait:               v.GetBool("wait"),
				Silent:             output != "",
			}

			stopCh := make(chan struct{})
			defer close(stopCh)

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
	cmd.Flags().Bool("disable-image-push", false, "set to true to disable images from being pushed to private registry")
	cmd.Flags().Bool("skip-registry-check", false, "set to true to skip the connectivity test and validation of the provided registry information")
	cmd.Flags().StringP("output", "o", "", "output format (currently supported: json)")

	cmd.Flags().Bool("debug", false, "when set, log full error traces in some cases where we provide a pretty message")
	cmd.Flags().MarkHidden("debug")

	registryFlags(cmd.Flags())

	return cmd
}

func logUpstreamUpgrade(log *logger.CLILogger, res *upstream.UpgradeResponse, output string) error {
	if output == "json" {
		outputJSON, err := json.Marshal(res)
		if err != nil {
			return errors.Wrap(err, "error marshaling JSON")
		}
		log.Info("%s", string(outputJSON))
		return nil
	}

	// text output
	if res.Error != "" {
		log.ActionWithoutSpinner("%s", res.Error)
	} else {
		if res.CurrentRelease != nil {
			log.ActionWithoutSpinner("Currently deployed release: sequence %v, version %v", res.CurrentRelease.Sequence, res.CurrentRelease.Version)
		}

		for _, r := range res.AvailableReleases {
			log.ActionWithoutSpinner("Downloading available release: sequence %v, version %v", r.Sequence, r.Version)
		}

		if res.DeployingRelease != nil {
			log.ActionWithoutSpinner("Deploying release: sequence %v, version %v", res.DeployingRelease.Sequence, res.DeployingRelease.Version)
		}
	}

	return nil
}
