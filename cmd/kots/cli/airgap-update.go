package cli

import (
	"bufio"
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
	registrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/image"
	imagetypes "github.com/replicatedhq/kots/pkg/image/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/tasks"
	"github.com/replicatedhq/kots/pkg/update"
	"github.com/replicatedhq/kots/pkg/upload"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func AirgapUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "airgap-update [appSlug]",
		Short:         "Process and upload an airgap update to the admin console",
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

			appSlug := args[0]
			log := logger.NewCLILogger(cmd.OutOrStdout())

			airgapBundle := v.GetString("airgap-bundle")
			if airgapBundle == "" {
				return fmt.Errorf("--airgap-bundle is required")
			}

			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			registryConfig, err := getRegistryConfig(v, clientset, appSlug)
			if err != nil {
				return errors.Wrap(err, "failed to get registry config")
			}

			pushOpts := imagetypes.PushImagesOptions{
				KotsadmTag: v.GetString("kotsadm-tag"),
				Registry: registrytypes.RegistryOptions{
					Endpoint:  registryConfig.OverrideRegistry,
					Namespace: registryConfig.OverrideNamespace,
					Username:  registryConfig.Username,
					Password:  registryConfig.Password,
				},
				ProgressWriter: getProgressWriter(v, log),
				LogForUI:       v.GetBool("from-api"),
			}

			if _, err := os.Stat(airgapBundle); err == nil {
				err = image.TagAndPushImagesFromBundle(airgapBundle, pushOpts)
				if err != nil {
					return errors.Wrap(err, "failed to push images")
				}
			} else {
				return errors.Wrap(err, "failed to stat airgap bundle")
			}

			updateFiles := []string{
				"airgap.yaml",
				"app.tar.gz",
			}
			if util.IsEmbeddedCluster() {
				updateFiles = append(updateFiles, "embedded-cluster/artifacts/kots.tar.gz")
			}

			airgapUpdate, err := archives.FilterAirgapBundle(airgapBundle, updateFiles)
			if err != nil {
				return errors.Wrap(err, "failed to create filtered airgap bundle")
			}
			defer os.RemoveAll(airgapUpdate)

			// we don't need to upload if we're already running in the api.
			// we can just register the airgap update directly.
			if v.GetBool("from-api") {
				if err := update.RegisterAirgapUpdateInDir(appSlug, airgapUpdate, v.GetString("updates-dir")); err != nil {
					return errors.Wrap(err, "failed to register airgap update")
				}
				return nil
			}

			stopCh := make(chan struct{})
			defer close(stopCh)

			localPort, errChan, err := upload.StartPortForward(namespace, stopCh, log)
			if err != nil {
				return err
			}
			uploadEndpoint := fmt.Sprintf("http://localhost:%d/api/v1/app/%s/airgap/update", localPort, url.PathEscape(appSlug))

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

			log.ActionWithSpinner("Uploading airgap update")
			if err := uploadAirgapUpdate(airgapUpdate, uploadEndpoint, namespace); err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to upload airgap update")
			}
			log.FinishSpinner()

			return nil
		},
	}

	cmd.Flags().StringP("namespace", "n", "", "the namespace in which kots/kotsadm is installed")
	cmd.Flags().String("airgap-bundle", "", "path to the application airgap bundle to upload")
	cmd.Flags().String("updates-dir", "", "path to the directory where updates are stored (only used when --from-api is set)")
	cmd.Flags().Bool("from-api", false, "whether the airgap update command was triggered by the API")

	registryFlags(cmd.Flags())

	return cmd
}

func getProgressWriter(v *viper.Viper, log *logger.CLILogger) io.Writer {
	if v.GetBool("from-api") {
		pipeReader, pipeWriter := io.Pipe()
		go func() {
			scanner := bufio.NewScanner(pipeReader)
			for scanner.Scan() {
				if err := tasks.SetTaskStatus("update-download", scanner.Text(), "running"); err != nil {
					log.Error(err)
				}
			}
			pipeReader.CloseWithError(scanner.Err())
		}()
		return pipeWriter
	}
	return os.Stdout
}

func uploadAirgapUpdate(airgapBundle string, uploadEndpoint string, namespace string) error {
	buffer := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(buffer)

	part, err := writer.CreateFormFile("application.airgap", "application.airgap")
	if err != nil {
		return errors.Wrap(err, "failed to create form file")
	}

	f, err := os.Open(airgapBundle)
	if err != nil {
		return errors.Wrap(err, "failed to open airgap bundle")
	}
	defer f.Close()

	if _, err := io.Copy(part, f); err != nil {
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

	newReq, err := util.NewRequest("PUT", uploadEndpoint, buffer)
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

	if resp.StatusCode == 404 {
		return errors.New("App not found")
	} else if resp.StatusCode != 200 {
		return errors.Errorf("Unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
