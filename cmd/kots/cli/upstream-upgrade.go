package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upload"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
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
			var images []kustomizetypes.Image

			isKurl, err := kotsadm.IsKurl(kubernetesConfigFlags)
			if err != nil {
				return errors.Wrap(err, "failed to check kURL")
			}

			airgapPath := ""
			if v.GetString("airgap-bundle") != "" {
				airgapRootDir, err := ioutil.TempDir("", "kotsadm-airgap")
				if err != nil {
					return errors.Wrap(err, "failed to create temp dir")
				}
				defer os.RemoveAll(airgapRootDir)

				registryEndpoint := v.GetString("kotsadm-registry")
				registryNamespace := v.GetString("kotsadm-namespace")
				registryUsername := v.GetString("registry-username")
				registryPassword := v.GetString("registry-password")

				if registryNamespace == "" {
					if isKurl {
						registryNamespace = appSlug
					} else {
						return errors.New("--kotsadm-namespace is required")
					}
				}

				if registryEndpoint == "" && isKurl {
					registryEndpoint, registryUsername, registryPassword, err = kotsutil.GetKurlRegistryCreds()
					if err != nil {
						return errors.Wrap(err, "failed to get kURL registry info")
					}
				}

				airgapPath = airgapRootDir

				err = kotsadm.ExtractAirgapImages(v.GetString("airgap-bundle"), airgapRootDir, os.Stdout)
				if err != nil {
					return errors.Wrap(err, "failed to extract images")
				}

				pushOptions := kotsadmtypes.PushImagesOptions{
					Registry: registry.RegistryOptions{
						Endpoint:  registryEndpoint,
						Namespace: registryNamespace,
						Username:  registryUsername,
						Password:  registryPassword,
					},
					ProgressWriter: os.Stdout,
				}

				imagesRootDir := filepath.Join(airgapRootDir, "images")
				images, err = kotsadm.TagAndPushAppImagesFromPath(imagesRootDir, pushOptions)
				if err != nil {
					return errors.Wrap(err, "failed to list image formats")
				}
			}

			log := logger.NewCLILogger()
			if airgapPath == "" {
				log.ActionWithSpinner("Checking for application updates")
			} else {
				log.ActionWithSpinner("Uploading application update")
			}

			stopCh := make(chan struct{})
			defer close(stopCh)
			localPort, errChan, err := upload.StartPortForward(v.GetString("namespace"), kubernetesConfigFlags, stopCh, log)
			if err != nil {
				log.FinishSpinnerWithError()
				return err
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

			contentType := "application/json"

			var requestBody io.Reader
			if airgapPath == "" {
				requestBody = strings.NewReader("{}")
			} else {
				buffer := &bytes.Buffer{}
				writer := multipart.NewWriter(buffer)

				if err := createPartFromFile(writer, airgapPath, "airgap.yaml"); err != nil {
					return errors.Wrap(err, "failed to create part from airgap.yaml")
				}
				if err := createPartFromFile(writer, airgapPath, "app.tar.gz"); err != nil {
					return errors.Wrap(err, "failed to create part from app.tar.gz")
				}

				b, err := json.Marshal(images)
				if err != nil {
					return errors.Wrap(err, "failed to marshal images data")
				}
				err = ioutil.WriteFile(filepath.Join(airgapPath, "images.json"), b, 0644)
				if err != nil {
					return errors.Wrap(err, "failed to write images data")
				}

				if err := createPartFromFile(writer, airgapPath, "images.json"); err != nil {
					return errors.Wrap(err, "failed to create part from images.json")
				}

				err = writer.Close()
				if err != nil {
					return errors.Wrap(err, "failed to close multi-part writer")
				}

				contentType = writer.FormDataContentType()
				requestBody = buffer
			}

			urlVals := url.Values{}
			if viper.GetBool("deploy") {
				urlVals.Set("deploy", "true")
			}
			if viper.GetBool("skip-preflights") {
				urlVals.Set("skipPreflights", "true")
			}

			updateCheckURI := fmt.Sprintf("http://localhost:%d/api/v1/app/%s/updatecheck?%s", localPort, url.PathEscape(appSlug), urlVals.Encode())

			authSlug, err := auth.GetOrCreateAuthSlug(kubernetesConfigFlags, v.GetString("namespace"))
			if err != nil {
				log.FinishSpinnerWithError()
				log.Info("Unable to authenticate to the Admin Console running in the %s namespace. Ensure you have read access to secrets in this namespace and try again.", v.GetString("namespace"))
				if v.GetBool("debug") {
					return errors.Wrap(err, "failed to get kotsadm auth slug")
				}
				os.Exit(2) // not returning error here as we don't want to show the entire stack trace to normal users
			}

			newReq, err := http.NewRequest("POST", updateCheckURI, requestBody)
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to create update check request")
			}
			newReq.Header.Add("Content-Type", contentType)
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

			if resp.StatusCode == 404 {
				log.FinishSpinnerWithError()
				return errors.Errorf("The application %s was not found in the cluster in the specified namespace", args[0])
			} else if resp.StatusCode != 200 {
				log.FinishSpinnerWithError()
				if len(b) != 0 {
					log.Error(errors.New(string(b)))
				}
				return errors.Errorf("Unexpected response from the API: %d", resp.StatusCode)
			}

			type updateCheckResponse struct {
				AvailableUpdates int `json:"availableUpdates"`
			}
			ucr := updateCheckResponse{}
			if err := json.Unmarshal(b, &ucr); err != nil {
				return errors.Wrap(err, "failed to parse response")
			}

			log.FinishSpinner()

			if viper.GetBool("deploy") {
				if airgapPath != "" {
					log.ActionWithoutSpinner("")
					log.ActionWithoutSpinner("Update has been uploaded and is being deployed")
					return nil
				}

				if ucr.AvailableUpdates == 0 {
					log.ActionWithoutSpinner("")
					log.ActionWithoutSpinner("There are no application updates available, ensuring latest is marked as deployed")
				} else {
					log.ActionWithoutSpinner("")
					log.ActionWithoutSpinner(fmt.Sprintf("There are currently %d updates available in the Admin Console, when the latest release is downloaded, it will be deployed", ucr.AvailableUpdates))
				}

				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner("To access the Admin Console, run kubectl kots admin-console --namespace %s", v.GetString("namespace"))
				log.ActionWithoutSpinner("")

				return nil
			}

			if airgapPath != "" {
				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner("Update has been uploaded")
			} else {
				if ucr.AvailableUpdates == 0 {
					log.ActionWithoutSpinner("")
					log.ActionWithoutSpinner("There are no application updates available")
				} else {
					log.ActionWithoutSpinner("")
					log.ActionWithoutSpinner(fmt.Sprintf("There are currently %d updates available in the Admin Console", ucr.AvailableUpdates))
				}
			}

			if !isKurl {
				log.ActionWithoutSpinner("To access the Admin Console, run kubectl kots admin-console --namespace %s", v.GetString("namespace"))
				log.ActionWithoutSpinner("")
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

	cmd.Flags().Bool("debug", false, "when set, log full error traces in some cases where we provide a pretty message")
	cmd.Flags().MarkHidden("debug")

	return cmd
}

func createPartFromFile(partWriter *multipart.Writer, path string, fileName string) error {
	file, err := os.Open(filepath.Join(path, fileName))
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}
	defer file.Close()

	part, err := partWriter.CreateFormFile(fileName, fileName)
	if err != nil {
		return errors.Wrap(err, "failed to create form file")
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return errors.Wrap(err, "failed to copy file to upload")
	}

	return nil
}
