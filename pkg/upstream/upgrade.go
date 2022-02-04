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
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
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
	RegistryEndpoint    string
	RegistryNamespace   string
	RegistryUsername    string
	RegistryPassword    string
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
	log := logger.NewCLILogger()
	if options.Silent {
		log.Silence()
	}

	airgapPath := ""
	var images []kustomizetypes.Image
	if options.AirgapBundle != "" {
		airgapRootDir, err := ioutil.TempDir("", "kotsadm-airgap")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create temp dir")
		}
		defer os.RemoveAll(airgapRootDir)

		registryEndpoint := options.RegistryEndpoint
		registryNamespace := options.RegistryNamespace
		registryUsername := options.RegistryUsername
		registryPassword := options.RegistryPassword

		if registryNamespace == "" {
			// check if it's provided as part of the registry endpoint
			parts := strings.Split(registryEndpoint, "/")
			if len(parts) > 1 {
				registryEndpoint = parts[0]
				registryNamespace = strings.Join(parts[1:], "/")
			}
		}

		if registryNamespace == "" {
			if options.IsKurl {
				registryNamespace = appSlug
			}
		}

		if registryEndpoint == "" && options.IsKurl {
			registryEndpoint, registryUsername, registryPassword, err = kotsutil.GetKurlRegistryCreds()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get kURL registry info")
			}
		}

		airgapPath = airgapRootDir

		err = kotsadm.ExtractAppAirgapArchive(options.AirgapBundle, airgapRootDir, options.DisableImagePush, os.Stdout)
		if err != nil {
			return nil, errors.Wrap(err, "failed to extract images")
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

		if options.DisableImagePush {
			images, err = kotsadm.GetImagesFromBundle(options.AirgapBundle, pushOptions)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get images from bundle")
			}
		} else {
			imagesRootDir := filepath.Join(airgapRootDir, "images")
			images, err = kotsadm.TagAndPushAppImagesFromPath(imagesRootDir, pushOptions)
			if err != nil {
				return nil, errors.Wrap(err, "failed to list image formats")
			}
		}
	}

	if airgapPath == "" {
		log.ActionWithSpinner("Checking for application updates")
	} else {
		log.ActionWithSpinner("Uploading application update")
	}

	contentType := "application/json"

	var requestBody io.Reader
	if airgapPath == "" {
		requestBody = strings.NewReader("{}")
	} else {
		buffer := &bytes.Buffer{}
		writer := multipart.NewWriter(buffer)

		if err := createPartFromFile(writer, airgapPath, "airgap.yaml"); err != nil {
			return nil, errors.Wrap(err, "failed to create part from airgap.yaml")
		}
		if err := createPartFromFile(writer, airgapPath, "app.tar.gz"); err != nil {
			return nil, errors.Wrap(err, "failed to create part from app.tar.gz")
		}

		b, err := json.Marshal(images)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal images data")
		}
		err = ioutil.WriteFile(filepath.Join(airgapPath, "images.json"), b, 0644)
		if err != nil {
			return nil, errors.Wrap(err, "failed to write images data")
		}

		if err := createPartFromFile(writer, airgapPath, "images.json"); err != nil {
			return nil, errors.Wrap(err, "failed to create part from images.json")
		}

		err = writer.Close()
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

	newReq, err := http.NewRequest("POST", options.UpdateCheckEndpoint, requestBody)
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
		if airgapPath != "" {
			log.ActionWithoutSpinner("")
			log.ActionWithoutSpinner("Update has been uploaded and is being deployed")
			return &ur, nil
		}

		if ur.AvailableUpdates == 0 {
			log.ActionWithoutSpinner("")
			if options.Deploy {
				log.ActionWithoutSpinner("There are no application updates available, ensuring latest is marked as deployed")
			} else {
				log.ActionWithoutSpinner("There are no application updates available, ensuring %s is marked as deployed", options.DeployVersionLabel)
			}
		} else if options.Wait {
			if options.Deploy {
				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner(fmt.Sprintf("There are currently %d updates available in the Admin Console, ensuring latest is marked as deployed", ur.AvailableUpdates))
			} else {
				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner(fmt.Sprintf("There are currently %d updates available in the Admin Console, ensuring %s is marked as deployed", ur.AvailableUpdates, options.DeployVersionLabel))
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

	if airgapPath != "" {
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
