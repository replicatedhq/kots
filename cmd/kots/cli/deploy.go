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

func DeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "deploy [appSlug]",
		Hidden: true, // Hidden from help
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

			log.ActionWithoutSpinner("Starting deployment process for %s...", appSlug)

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

			// Step 1: License Sync
			if err := handleLicenseSync(v, appSlug, localPort, authSlug, log); err != nil {
				return errors.Wrap(err, "failed to sync license")
			}

			// Step 2: Download Upstream Update
			if err := handleDownloadUpstreamUpdate(v, appSlug, localPort, authSlug, log); err != nil {
				return errors.Wrap(err, "failed to download upstream update")
			}

			// Step 3: Set Config + Deploy
			if err := handleSetConfigAndDeploy(v, appSlug, localPort, authSlug, log); err != nil {
				return errors.Wrap(err, "failed to set config and deploy")
			}

			log.ActionWithoutSpinner("Deployment process completed successfully")

			return nil
		},
	}

	cmd.Flags().String("license", "", "path to license file (optional)")
	cmd.Flags().String("channel-id", "", "channel ID")
	cmd.Flags().Int64("channel-sequence", 0, "channel sequence")
	cmd.Flags().String("config-values", "", "path to config values file")
	cmd.Flags().Bool("skip-preflights", false, "skip preflight checks")

	cmd.MarkFlagRequired("channel-id")
	cmd.MarkFlagRequired("channel-sequence")
	cmd.MarkFlagRequired("config-values")

	return cmd
}

func handleLicenseSync(v *viper.Viper, appSlug string, localPort int, authSlug string, log *logger.CLILogger) error {
	log.ActionWithoutSpinner("Syncing license...")

	licenseData := ""

	// Check if license flag was provided
	if licenseFilePath := v.GetString("license"); licenseFilePath != "" {
		data, err := os.ReadFile(licenseFilePath)
		if err != nil {
			return errors.Wrap(err, "failed to read license file")
		}
		licenseData = string(data)
	}
	// If no license provided, licenseData is empty and sync uses current license

	requestPayload := map[string]interface{}{
		"licenseData": licenseData,
	}

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return errors.Wrap(err, "failed to marshal license sync request json")
	}

	url := fmt.Sprintf("http://localhost:%d/api/v1/app/%s/license", localPort, url.QueryEscape(appSlug))
	newRequest, err := http.NewRequest("PUT", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return errors.Wrap(err, "failed to create license sync http request")
	}
	newRequest.Header.Add("Authorization", authSlug)
	newRequest.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(newRequest)
	if err != nil {
		return errors.Wrap(err, "failed to execute license sync http request")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read license sync server response")
	}

	response := struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
		Synced  bool   `json:"synced"`
	}{}
	_ = json.Unmarshal(respBody, &response)

	if resp.StatusCode != http.StatusOK {
		if response.Error != "" {
			return errors.Errorf("license sync failed: %s", response.Error)
		}
		return errors.Errorf("license sync failed with status code %d", resp.StatusCode)
	}

	if response.Synced {
		log.ActionWithoutSpinner("License synced successfully")
	} else {
		log.ActionWithoutSpinner("License already up to date")
	}

	return nil
}

func handleDownloadUpstreamUpdate(v *viper.Viper, appSlug string, localPort int, authSlug string, log *logger.CLILogger) error {
	log.ActionWithoutSpinner("Downloading upstream update...")

	channelID := v.GetString("channel-id")
	channelSequence := v.GetInt64("channel-sequence")
	skipPreflights := v.GetBool("skip-preflights")

	if channelID == "" {
		return errors.New("channel-id is required")
	}
	if channelSequence == 0 {
		return errors.New("channel-sequence is required")
	}

	requestPayload := map[string]interface{}{
		"channelId":       channelID,
		"channelSequence": channelSequence,
		"skipPreflights":  skipPreflights,
	}

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return errors.Wrap(err, "failed to marshal upstream update request json")
	}

	url := fmt.Sprintf("http://localhost:%d/api/v1/app/%s/upstream/download", localPort, url.QueryEscape(appSlug))
	newRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return errors.Wrap(err, "failed to create upstream update http request")
	}
	newRequest.Header.Add("Authorization", authSlug)
	newRequest.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(newRequest)
	if err != nil {
		return errors.Wrap(err, "failed to execute upstream update http request")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read upstream update server response")
	}

	response := struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}{}
	_ = json.Unmarshal(respBody, &response)

	if resp.StatusCode != http.StatusOK {
		if response.Error != "" {
			return errors.Errorf("upstream update failed: %s", response.Error)
		}
		return errors.Errorf("upstream update failed with status code %d", resp.StatusCode)
	}

	log.ActionWithoutSpinner("Upstream update downloaded successfully")

	return nil
}

func handleSetConfigAndDeploy(v *viper.Viper, appSlug string, localPort int, authSlug string, log *logger.CLILogger) error {
	log.ActionWithoutSpinner("Setting configuration and deploying...")

	// Read config values from file (required flag)
	configValues, err := os.ReadFile(v.GetString("config-values"))
	if err != nil {
		return errors.Wrap(err, "failed to read config values file")
	}

	requestPayload := map[string]interface{}{
		"configValues":   configValues,
		"merge":          false, // Always false for deploy command
		"deploy":         true,  // HARDCODED - always deploy
		"skipPreflights": v.GetBool("skip-preflights"),
		"current":        false,
		"sequence":       int64(-1),
	}

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return errors.Wrap(err, "failed to marshal config request json")
	}

	url := fmt.Sprintf("http://localhost:%d/api/v1/app/%s/config/values", localPort, url.QueryEscape(appSlug))
	newRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return errors.Wrap(err, "failed to create config http request")
	}
	newRequest.Header.Add("Authorization", authSlug)
	newRequest.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(newRequest)
	if err != nil {
		return errors.Wrap(err, "failed to execute config http request")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read config server response")
	}

	response := struct {
		Error string `json:"error"`
	}{}
	_ = json.Unmarshal(respBody, &response)

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return errors.Errorf("app with slug %s not found", appSlug)
		} else if response.Error != "" {
			return errors.New(response.Error)
		} else {
			return errors.Errorf("config and deploy failed with status code %d", resp.StatusCode)
		}
	}

	log.ActionWithoutSpinner("Configuration set and deployment initiated successfully")

	return nil
}
