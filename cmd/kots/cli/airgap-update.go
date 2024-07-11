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
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/tasks"
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

			updateFiles, err := getAirgapUpdateFiles(airgapBundle)
			if err != nil {
				return errors.Wrap(err, "failed to get airgap update files")
			}
			airgapUpdate, err := archives.FilterAirgapBundle(airgapBundle, updateFiles)
			if err != nil {
				return errors.Wrap(err, "failed to create filtered airgap bundle")
			}
			defer os.RemoveAll(airgapUpdate)

			var localPort int
			if v.GetBool("from-api") {
				localPort = 3000
			} else {
				stopCh := make(chan struct{})
				defer close(stopCh)

				lp, errChan, err := upload.StartPortForward(namespace, stopCh, log)
				if err != nil {
					return err
				}
				localPort = lp

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
			}

			uploadEndpoint := fmt.Sprintf("http://localhost:%d/api/v1/app/%s/airgap/update", localPort, url.PathEscape(appSlug))

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

	cmd.Flags().Bool("from-api", false, "whether the airgap update command was triggered by the API")
	cmd.Flags().String("task-id", "", "the task ID to use for tracking progress")
	cmd.Flags().MarkHidden("from-api")
	cmd.Flags().MarkHidden("task-id")

	registryFlags(cmd.Flags())

	return cmd
}

func getProgressWriter(v *viper.Viper, log *logger.CLILogger) io.Writer {
	if v.GetBool("from-api") {
		pipeReader, pipeWriter := io.Pipe()
		go func() {
			scanner := bufio.NewScanner(pipeReader)
			for scanner.Scan() {
				if err := tasks.SetTaskStatus(v.GetString("task-id"), scanner.Text(), "running"); err != nil {
					log.Error(err)
				}
			}
			pipeReader.CloseWithError(scanner.Err())
		}()
		return pipeWriter
	}
	return os.Stdout
}

func getAirgapUpdateFiles(airgapBundle string) ([]string, error) {
	airgap, err := kotsutil.FindAirgapMetaInBundle(airgapBundle)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find airgap meta in bundle")
	}

	if airgap.Spec.EmbeddedClusterArtifacts == nil {
		return nil, errors.New("embedded cluster artifacts not found in airgap bundle")
	}

	if airgap.Spec.EmbeddedClusterArtifacts.Metadata == "" {
		return nil, errors.New("embedded cluster metadata not found in airgap bundle")
	}

	if airgap.Spec.EmbeddedClusterArtifacts.AdditionalArtifacts == nil {
		return nil, errors.New("embedded cluster additional artifacts not found in airgap bundle")
	}

	files := []string{
		"airgap.yaml",
		"app.tar.gz",
		airgap.Spec.EmbeddedClusterArtifacts.Metadata,
		airgap.Spec.EmbeddedClusterArtifacts.AdditionalArtifacts["kots"],
	}

	return files, nil
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
