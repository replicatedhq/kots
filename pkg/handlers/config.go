package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/multitype"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kotsadm/pkg/app"
	"github.com/replicatedhq/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/session"
)

type UpdateAppConfigRequest struct {
	Sequence         int                        `json:"sequence"`
	CreateNewVersion bool                       `json:"createNewVersion"`
	ConfigGroups     []*kotsv1beta1.ConfigGroup `json:"configGroups"`
}

type UpdateAppConfigResponse struct {
	Success       bool     `json:"success"`
	Error         string   `json:"error,omitempty"`
	RequiredItems []string `json:"requiredItems,omitempty"`
}

func UpdateAppConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	updateAppConfigResponse := UpdateAppConfigResponse{
		Success: false,
	}

	updateAppConfigRequest := UpdateAppConfigRequest{}
	if err := json.NewDecoder(r.Body).Decode(&updateAppConfigRequest); err != nil {
		logger.Error(err)
		updateAppConfigResponse.Error = "failed to decode request body"
		JSON(w, 400, updateAppConfigResponse)
		return
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		updateAppConfigResponse.Error = "failed to parse authorization header"
		JSON(w, 401, updateAppConfigResponse)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		updateAppConfigResponse.Error = "failed to parse authorization header"
		JSON(w, 401, updateAppConfigResponse)
		return
	}

	foundApp, err := app.GetFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		updateAppConfigResponse.Error = "failed to get app from app slug"
		JSON(w, 500, updateAppConfigResponse)
		return
	}

	archiveDir, err := app.GetAppVersionArchive(foundApp.ID, updateAppConfigRequest.Sequence)
	if err != nil {
		logger.Error(err)
		updateAppConfigResponse.Error = "failed to get app version archive"
		JSON(w, 500, updateAppConfigResponse)
		return
	}
	defer os.RemoveAll(archiveDir)

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		logger.Error(err)
		updateAppConfigResponse.Error = "failed to load kots kinds from path"
		JSON(w, 500, updateAppConfigResponse)
		return
	}

	// check for unset required items
	requiredItems := make([]string, 0, 0)
	requiredItemsTitles := make([]string, 0, 0)
	for _, group := range updateAppConfigRequest.ConfigGroups {
		for _, item := range group.Items {
			if app.IsRequiredItem(item) && app.IsUnsetItem(item) {
				requiredItems = append(requiredItems, item.Name)
				if item.Title != "" {
					requiredItemsTitles = append(requiredItemsTitles, item.Title)
				} else {
					requiredItemsTitles = append(requiredItemsTitles, item.Name)
				}
			}
		}
	}

	if len(requiredItems) > 0 {
		updateAppConfigResponse := UpdateAppConfigResponse{
			Success:       false,
			Error:         fmt.Sprintf("The following fields are required: %s", strings.Join(requiredItemsTitles, ", ")),
			RequiredItems: requiredItems,
		}
		JSON(w, 400, updateAppConfigResponse)
		return
	}

	// we don't merge, this is a wholesale replacement of the config values
	// so we don't need the complex logic in kots, we can just write
	values := kotsKinds.ConfigValues.Spec.Values
	for _, group := range updateAppConfigRequest.ConfigGroups {
		for _, item := range group.Items {
			if item.Value.Type == multitype.Bool {
				updatedValue := item.Value.BoolVal
				v := values[item.Name]
				v.Value = strconv.FormatBool(updatedValue)
				values[item.Name] = v
			} else if item.Value.Type == multitype.String {
				updatedValue := item.Value.String()
				if item.Type == "password" {
					// encrypt using the key
					cipher, err := crypto.AESCipherFromString(kotsKinds.Installation.Spec.EncryptionKey)
					if err != nil {
						logger.Error(err)
						updateAppConfigResponse.Error = "failed to get encryption cipher"
						JSON(w, 500, updateAppConfigResponse)
						return
					}

					// if the decryption succeeds, don't encrypt again
					_, err = decrypt(updatedValue, cipher)
					if err != nil {
						updatedValue = base64.StdEncoding.EncodeToString(cipher.Encrypt([]byte(updatedValue)))
					}
				}

				v := values[item.Name]
				v.Value = updatedValue
				values[item.Name] = v
			}
		}
	}

	if kotsKinds.ConfigValues == nil {
		logger.Error(errors.New("no config values found"))
		updateAppConfigResponse.Error = "no config values found"
		JSON(w, 500, updateAppConfigResponse)
		return
	}

	kotsKinds.ConfigValues.Spec.Values = values

	configValuesSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "ConfigValues")
	if err != nil {
		logger.Error(err)
		updateAppConfigResponse.Error = "failed to marshal config values spec"
		JSON(w, 500, updateAppConfigResponse)
		return
	}

	if err := ioutil.WriteFile(filepath.Join(archiveDir, "upstream", "userdata", "config.yaml"), []byte(configValuesSpec), 0644); err != nil {
		logger.Error(err)
		updateAppConfigResponse.Error = "failed to write config.yaml to upstream/userdata"
		JSON(w, 500, updateAppConfigResponse)
		return
	}

	err = foundApp.RenderDir(archiveDir)
	if err != nil {
		logger.Error(err)
		updateAppConfigResponse.Error = "failed to render archive directory"
		JSON(w, 500, updateAppConfigResponse)
		return
	}

	if updateAppConfigRequest.CreateNewVersion {
		newSequence, err := foundApp.CreateVersion(archiveDir, "Config Change")
		if err != nil {
			logger.Error(err)
			updateAppConfigResponse.Error = "failed to create an app version"
			JSON(w, 500, updateAppConfigResponse)
			return
		}

		if err := app.CreateAppVersionArchive(foundApp.ID, newSequence, archiveDir); err != nil {
			logger.Error(err)
			updateAppConfigResponse.Error = "failed to create an app version archive"
			JSON(w, 500, updateAppConfigResponse)
			return
		}
	} else {
		if err := app.UpdateConfigValuesInDB(archiveDir, foundApp.ID, int64(updateAppConfigRequest.Sequence)); err != nil {
			logger.Error(err)
			updateAppConfigResponse.Error = "failed to update config values in db"
			JSON(w, 500, updateAppConfigResponse)
			return
		}

		if kotsKinds.Preflight != nil {
			if err := app.SetDownstreamVersionPendingPreflight(foundApp.ID, int64(updateAppConfigRequest.Sequence)); err != nil {
				logger.Error(err)
				updateAppConfigResponse.Error = "failed to set downstream version pending preflight"
				JSON(w, 500, updateAppConfigResponse)
				return
			}
		} else {
			if err := app.SetDownstreamVersionReady(foundApp.ID, int64(updateAppConfigRequest.Sequence)); err != nil {
				logger.Error(err)
				updateAppConfigResponse.Error = "failed to set downstream version ready"
				JSON(w, 500, updateAppConfigResponse)
				return
			}
		}

		if err := app.CreateAppVersionArchive(foundApp.ID, int64(updateAppConfigRequest.Sequence), archiveDir); err != nil {
			logger.Error(err)
			updateAppConfigResponse.Error = "failed to create app version archive"
			JSON(w, 500, updateAppConfigResponse)
			return
		}
	}

	updateAppConfigResponse.Success = true

	JSON(w, 200, updateAppConfigResponse)
}

func decrypt(input string, cipher *crypto.AESCipher) (string, error) {
	if cipher == nil {
		return "", errors.New("cipher not defined")
	}

	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", errors.Wrap(err, "failed to base64 decode")
	}

	decrypted, err := cipher.Decrypt(decoded)
	if err != nil {
		return "", errors.Wrap(err, "failed to decrypt")
	}

	return string(decrypted), nil
}
