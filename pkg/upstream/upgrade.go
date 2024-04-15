package upstream

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/archives"
	"github.com/replicatedhq/kots/pkg/auth"
	registrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/image"
	imagetypes "github.com/replicatedhq/kots/pkg/image/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
)

type UpgradeResponse struct {
	Success           bool             `json:"success"`
	AvailableUpdates  int64            `json:"availableUpdates"`
	CurrentRelease    *UpgradeRelease  `json:"currentRelease,omitempty"`
	AvailableReleases []UpgradeRelease `json:"availableReleases,omitempty"`
	DeployingRelease  *UpgradeRelease  `json:"deployingRelease,omitempty"`
	Error             string           `json:"error,omitempty"`
}

type UpgradeRelease struct {
	Sequence int64  `json:"sequence"`
	Version  string `json:"version"`
}

type UpgradeOptions struct {
	AirgapBundle        string
	RegistryConfig      kotsadmtypes.RegistryConfig
	IsKurl              bool
	DisableImagePush    bool
	UpdateCheckEndpoint string
	Namespace           string
	Debug               bool
	Deploy              bool
	DeployVersionLabel  string
	Wait                bool
	Silent              bool
}

func Upgrade(appSlug string, options UpgradeOptions) (*UpgradeResponse, error) {
	log := logger.NewCLILogger(os.Stdout)
	if options.Silent {
		log.Silence()
	}

	if options.AirgapBundle != "" {
		pushOptions := imagetypes.PushImagesOptions{
			Registry: registrytypes.RegistryOptions{
				Endpoint:  options.RegistryConfig.OverrideRegistry,
				Namespace: options.RegistryConfig.OverrideNamespace,
				Username:  options.RegistryConfig.Username,
				Password:  options.RegistryConfig.Password,
			},
			ProgressWriter: os.Stdout,
		}

		if !options.DisableImagePush {
			err := image.TagAndPushImagesFromBundle(options.AirgapBundle, pushOptions)
			if err != nil {
				return nil, errors.Wrap(err, "failed to tag and push app images from path")
			}
		}
	}

	if options.AirgapBundle == "" {
		log.ActionWithSpinner("Checking for application updates")
	} else {
		log.ActionWithSpinner("Uploading application update")
	}

	contentType := "application/json"

	var requestBody io.Reader
	if options.AirgapBundle == "" {
		requestBody = strings.NewReader("{}")
	} else {
		buffer := &bytes.Buffer{}
		writer := multipart.NewWriter(buffer)

		if err := createPartFromFile(writer, options.AirgapBundle, "airgap.yaml"); err != nil {
			return nil, errors.Wrap(err, "failed to create part from airgap.yaml")
		}
		if err := createPartFromFile(writer, options.AirgapBundle, "app.tar.gz"); err != nil {
			return nil, errors.Wrap(err, "failed to create part from app.tar.gz")
		}

		err := writer.Close()
		if err != nil {
			return nil, errors.Wrap(err, "failed to close multi-part writer")
		}

		contentType = writer.FormDataContentType()
		requestBody = buffer
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}

	authSlug, err := auth.GetOrCreateAuthSlug(clientset, options.Namespace)
	if err != nil {
		log.FinishSpinnerWithError()
		log.Info("Unable to authenticate to the Admin Console running in the %s namespace. Ensure you have read access to secrets in this namespace and try again.", options.Namespace)
		if options.Debug {
			return nil, errors.Wrap(err, "failed to get kotsadm auth slug")
		}
		os.Exit(2) // not returning error here as we don't want to show the entire stack trace to normal users
	}

	newReq, err := util.NewRequest("POST", options.UpdateCheckEndpoint, requestBody)
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to create update check request")
	}
	newReq.Header.Add("Content-Type", contentType)
	newReq.Header.Add("Authorization", authSlug)

	resp, err := http.DefaultClient.Do(newReq)
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to check for updates")
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to read server response")
	}

	if resp.StatusCode == 404 {
		log.FinishSpinnerWithError()
		return nil, errors.Errorf("The application %s was not found in the cluster in the specified namespace", appSlug)
	} else if resp.StatusCode != 200 {
		log.FinishSpinnerWithError()
		if len(b) != 0 {
			log.Error(errors.New(string(b)))
		}
		return nil, errors.Errorf("Unexpected response from the API: %d", resp.StatusCode)
	}

	ur := UpgradeResponse{}
	if err := json.Unmarshal(b, &ur); err != nil {
		return nil, errors.Wrap(err, "failed to parse response")
	}
	if ur.DeployingRelease != nil && ur.DeployingRelease.Version == "" {
		ur.DeployingRelease = nil
	}

	log.FinishSpinner()

	if options.Deploy || options.DeployVersionLabel != "" {
		if options.AirgapBundle != "" {
			log.ActionWithoutSpinner("")
			log.ActionWithoutSpinner("Update has been uploaded and is being deployed")
			return &ur, nil
		}

		if ur.AvailableUpdates == 0 {
			log.ActionWithoutSpinner("")
			if options.Deploy {
				log.ActionWithoutSpinner("There are no application updates available, ensuring latest is deployed")
			} else {
				log.ActionWithoutSpinner("There are no application updates available, ensuring %s is deployed", options.DeployVersionLabel)
			}
		} else if options.Wait {
			if options.Deploy {
				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner(fmt.Sprintf("There are currently %d updates available in the Admin Console, ensuring latest is deployed", ur.AvailableUpdates))
			} else {
				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner(fmt.Sprintf("There are currently %d updates available in the Admin Console, ensuring %s is deployed", ur.AvailableUpdates, options.DeployVersionLabel))
			}
		} else {
			if options.Deploy {
				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner(fmt.Sprintf("There are currently %d updates available in the Admin Console, when the latest release is downloaded, it will be deployed", ur.AvailableUpdates))
			} else {
				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner(fmt.Sprintf("There are currently %d updates available in the Admin Console, when the release with the %s version label is downloaded, it will be deployed", ur.AvailableUpdates, options.DeployVersionLabel))
			}
		}

		log.ActionWithoutSpinner("")
		log.ActionWithoutSpinner("To access the Admin Console, run kubectl kots admin-console --namespace %s", options.Namespace)
		log.ActionWithoutSpinner("")

		return &ur, nil
	}

	if options.AirgapBundle != "" {
		log.ActionWithoutSpinner("")
		log.ActionWithoutSpinner("Update has been uploaded")
	} else {
		if ur.AvailableUpdates == 0 {
			log.ActionWithoutSpinner("")
			log.ActionWithoutSpinner("There are no application updates available")
		} else {
			log.ActionWithoutSpinner("")
			log.ActionWithoutSpinner(fmt.Sprintf("There are currently %d updates available in the Admin Console", ur.AvailableUpdates))
		}
	}

	if !options.IsKurl {
		log.ActionWithoutSpinner("To access the Admin Console, run kubectl kots admin-console --namespace %s", options.Namespace)
		log.ActionWithoutSpinner("")
	}

	return &ur, nil
}

func createPartFromFile(partWriter *multipart.Writer, path string, fileName string) error {
	contents, err := archives.GetFileFromAirgap(fileName, path)
	if err != nil {
		return errors.Wrapf(err, "failed to get file %s from airgap", fileName)
	}

	part, err := partWriter.CreateFormFile(fileName, fileName)
	if err != nil {
		return errors.Wrap(err, "failed to create form file")
	}

	_, err = part.Write(contents)
	if err != nil {
		return errors.Wrap(err, "failed to write part")
	}

	return nil
}
