package cli

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/archives"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upload"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func AirgapUploadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "airgap-upload [appSlug]",
		Short:         "Upload an airgap bundle to the admin console",
		Long:          "",
		SilenceUsage:  true,
		SilenceErrors: false,
		Hidden:        true,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if len(args) == 0 {
				cmd.Help()
				os.Exit(1)
			}

			airgapBundle := v.GetString("airgap-bundle")
			if airgapBundle == "" {
				return fmt.Errorf("--airgap-bundle is required")
			}

			appSlug := args[0]
			log := logger.NewCLILogger(cmd.OutOrStdout())

			filesToInclude := []string{
				"airgap.yaml",
				"app.tar.gz",
				"embedded-cluster/artifacts/kots",
			}

			filteredAirgapBundle, err := archives.CreateFilteredAirgapBundle(airgapBundle, filesToInclude)
			if err != nil {
				return errors.Wrap(err, "failed to create filtered airgap bundle")
			}

			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}

			stopCh := make(chan struct{})
			defer close(stopCh)

			localPort, errChan, err := upload.StartPortForward(namespace, stopCh, log)
			if err != nil {
				return err
			}
			uploadEndpoint := fmt.Sprintf("http://localhost:%d/api/v1/app/%s/airgap-upload", localPort, url.PathEscape(appSlug))

			go func() {
				select {
				case err := <-errChan:
					if err != nil {
						log.Error(err)
						os.Exit(1)
					}
				case <-stopCh:
				}
			}()

			log.ActionWithSpinner("Uploading airgap bundle")
			if err := uploadAirgapBundle(filteredAirgapBundle, uploadEndpoint, namespace); err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to upload airgap bundle")
			}
			log.FinishSpinner()

			return nil
		},
	}

	cmd.Flags().String("airgap-bundle", "", "path to the application airgap bundle to upload")

	return cmd
}

func uploadAirgapBundle(airgapBundle io.Reader, uploadEndpoint string, namespace string) error {
	buffer := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(buffer)

	part, err := writer.CreateFormFile("application.airgap", "application.airgap")
	if err != nil {
		return errors.Wrap(err, "failed to create form file")
	}

	if _, err := io.Copy(part, airgapBundle); err != nil {
		return errors.Wrap(err, "failed to copy airgap bundle to form file")
	}

	err = writer.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close writer")
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s clientset")
	}

	authSlug, err := auth.GetOrCreateAuthSlug(clientset, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to get auth slug")
	}

	newReq, err := util.NewRequest("POST", uploadEndpoint, buffer)
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}
	newReq.Header.Add("Content-Type", writer.FormDataContentType())
	newReq.Header.Add("Authorization", authSlug)

	resp, err := http.DefaultClient.Do(newReq)
	if err != nil {
		return errors.Wrap(err, "failed to make request")
	}
	defer resp.Body.Close()

	// b, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	return errors.Wrap(err, "failed to read response body")
	// }

	if resp.StatusCode == 404 {
		return errors.New("App not found")
	} else if resp.StatusCode != 200 {
		return errors.Errorf("Unexpected status code: %d", resp.StatusCode)
	}

	// // TODO: read the response body
	// ur := UpgradeResponse{}
	// if err := json.Unmarshal(b, &ur); err != nil {
	// 	return errors.Wrap(err, "failed to unmarshal response")
	// }

	return nil
}
