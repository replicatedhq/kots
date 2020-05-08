package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/multitype"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/config"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/preflight"
	"github.com/replicatedhq/kots/kotsadm/pkg/registry"
	"github.com/replicatedhq/kots/kotsadm/pkg/render"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	versiontypes "github.com/replicatedhq/kots/kotsadm/pkg/version/types"
)

type UpdateAppConfigRequest struct {
	Sequence         int64                      `json:"sequence"`
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

	if !updateAppConfigRequest.CreateNewVersion {
		// special case handling for "do not create new version"
		// no need to update versions after this/search for latest sequence that has the same upstream version/etc
		resp, err := updateAppConfig(foundApp, updateAppConfigRequest.Sequence, updateAppConfigRequest, true)
		if err != nil {
			logger.Error(err)
			JSON(w, 500, resp)
			return
		}

		if len(resp.RequiredItems) > 0 {
			JSON(w, 400, resp)
			return
		}

		JSON(w, 200, resp)
		return
	}

	// find the update cursor referred to by updateAppConfigRequest.Sequence
	// then find the latest sequence with that update cursor, and use that for the update we're making here
	// (for instance, the registry settings may have changed)
	// if there are additional update cursors past this, also create updates for them
	latestSequenceMatchingUpdateCursor, laterVersions, err := getLaterVersions(foundApp, updateAppConfigRequest.Sequence)
	if err != nil {
		logger.Error(err)
		updateAppConfigResponse.Error = err.Error()
		JSON(w, 500, updateAppConfigResponse)
		return
	}

	// attempt to apply the config to the app version specified in the request
	resp, err := updateAppConfig(foundApp, latestSequenceMatchingUpdateCursor, updateAppConfigRequest, true)
	if err != nil {
		logger.Error(err)
		JSON(w, 500, resp)
		return
	}

	if len(resp.RequiredItems) > 0 {
		JSON(w, 400, resp)
		return
	}

	// if there were no errors applying the config for the desired version, do the same for any later versions too
	for _, version := range laterVersions {
		_, err := updateAppConfig(foundApp, version.Sequence, updateAppConfigRequest, false)
		if err != nil {
			logger.Error(errors.Wrapf(err, "error creating app with new config based on sequence %d for upstream %q", version.Sequence, version.VersionLabel))
		}
	}

	JSON(w, 200, UpdateAppConfigResponse{Success: true})
}

// if isPrimaryVersion is false, missing a required config field will not cause a failure, and instead will create
// the app version with status needs_config
func updateAppConfig(updateApp *app.App, sequence int64, req UpdateAppConfigRequest, isPrimaryVersion bool) (UpdateAppConfigResponse, error) {
	updateAppConfigResponse := UpdateAppConfigResponse{
		Success: false,
	}

	archiveDir, err := version.GetAppVersionArchive(updateApp.ID, sequence)
	if err != nil {
		updateAppConfigResponse.Error = "failed to get app version archive"
		return updateAppConfigResponse, err
	}
	defer os.RemoveAll(archiveDir)

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		updateAppConfigResponse.Error = "failed to load kots kinds from path"
		return updateAppConfigResponse, err
	}

	// check for unset required items
	requiredItems := make([]string, 0, 0)
	requiredItemsTitles := make([]string, 0, 0)
	for _, group := range req.ConfigGroups {
		for _, item := range group.Items {
			if config.IsRequiredItem(item) && config.IsUnsetItem(item) {
				requiredItems = append(requiredItems, item.Name)
				if item.Title != "" {
					requiredItemsTitles = append(requiredItemsTitles, item.Title)
				} else {
					requiredItemsTitles = append(requiredItemsTitles, item.Name)
				}
			}
		}
	}

	// not having all the required items is only a failure for the version that the user intended to edit
	if len(requiredItems) > 0 && isPrimaryVersion {
		updateAppConfigResponse.RequiredItems = requiredItems
		updateAppConfigResponse.Error = fmt.Sprintf("The following fields are required: %s", strings.Join(requiredItemsTitles, ", "))
		return updateAppConfigResponse, nil
	}

	// we don't merge, this is a wholesale replacement of the config values
	// so we don't need the complex logic in kots, we can just write
	values := kotsKinds.ConfigValues.Spec.Values
	for _, group := range req.ConfigGroups {
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
						updateAppConfigResponse.Error = "failed to get encryption cipher"
						return updateAppConfigResponse, err
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
		updateAppConfigResponse.Error = "no config values found"
		return updateAppConfigResponse, errors.New("no config values found")
	}

	kotsKinds.ConfigValues.Spec.Values = values

	configValuesSpec, err := kotsKinds.Marshal("kots.io", "v1beta1", "ConfigValues")
	if err != nil {
		updateAppConfigResponse.Error = "failed to marshal config values spec"
		return updateAppConfigResponse, err
	}

	if err := ioutil.WriteFile(filepath.Join(archiveDir, "upstream", "userdata", "config.yaml"), []byte(configValuesSpec), 0644); err != nil {
		updateAppConfigResponse.Error = "failed to write config.yaml to upstream/userdata"
		return updateAppConfigResponse, err
	}

	appSequence := updateApp.CurrentSequence
	if req.CreateNewVersion {
		appSequence, err = version.GetNextAppSequence(updateApp.ID, &updateApp.CurrentSequence)
		if err != nil {
			updateAppConfigResponse.Error = "failed to get next app sequence"
			return updateAppConfigResponse, errors.Wrap(err, "failed to get new app sequence")
		}
	}

	registrySettings, err := registry.GetRegistrySettingsForApp(updateApp.ID)
	if err != nil {
		updateAppConfigResponse.Error = "failed to get registry settings"
		return updateAppConfigResponse, err
	}
	err = render.RenderDir(archiveDir, updateApp.ID, appSequence, registrySettings)
	if err != nil {
		updateAppConfigResponse.Error = "failed to render archive directory"
		return updateAppConfigResponse, err
	}

	if req.CreateNewVersion {
		newSequence, err := version.CreateVersion(updateApp.ID, archiveDir, "Config Change", updateApp.CurrentSequence)
		if err != nil {
			updateAppConfigResponse.Error = "failed to create an app version"
			return updateAppConfigResponse, err
		}
		sequence = newSequence

		if err := version.CreateAppVersionArchive(updateApp.ID, newSequence, archiveDir); err != nil {
			updateAppConfigResponse.Error = "failed to create an app version archive"
			return updateAppConfigResponse, err
		}
	} else {
		if err := config.UpdateConfigValuesInDB(archiveDir, updateApp.ID, int64(sequence)); err != nil {
			updateAppConfigResponse.Error = "failed to update config values in db"
			return updateAppConfigResponse, err
		}

		if err := version.CreateAppVersionArchive(updateApp.ID, int64(sequence), archiveDir); err != nil {
			updateAppConfigResponse.Error = "failed to create app version archive"
			return updateAppConfigResponse, err
		}
	}

	if err := downstream.SetDownstreamVersionPendingPreflight(updateApp.ID, int64(sequence)); err != nil {
		updateAppConfigResponse.Error = "failed to set downstream status to 'pending preflight'"
		return updateAppConfigResponse, err
	}

	if err := preflight.Run(updateApp.ID, int64(sequence), archiveDir); err != nil {
		updateAppConfigResponse.Error = errors.Cause(err).Error()
		return updateAppConfigResponse, err
	}

	updateAppConfigResponse.Success = true
	return updateAppConfigResponse, nil
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

func getLaterVersions(versionedApp *app.App, startSequence int64) (int64, []versiontypes.AppVersion, error) {
	versions, err := version.GetVersions(versionedApp.ID)
	if err != nil {
		return -1, nil, errors.Wrap(err, "failed to get app versions")
	}

	thisUpdateCursor := -1
	latestSequenceWithUpdateCursor := startSequence
	laterVersions := map[int]versiontypes.AppVersion{}
	for _, version := range versions {
		if version.Sequence == startSequence {
			thisUpdateCursor = version.UpdateCursor
		}
	}
	if thisUpdateCursor == -1 {
		err := fmt.Errorf("unable to find update cursor for sequence %d in %+v", startSequence, versions)
		return -1, nil, err
	}
	for _, version := range versions {
		if version.UpdateCursor == thisUpdateCursor && version.Sequence > latestSequenceWithUpdateCursor {
			latestSequenceWithUpdateCursor = version.Sequence
		}
		if version.UpdateCursor > thisUpdateCursor {
			// save the latest sequence # for a given update cursor
			ver, ok := laterVersions[version.UpdateCursor]
			if !ok {
				laterVersions[version.UpdateCursor] = version
			} else if ver.Sequence < version.Sequence {
				laterVersions[version.UpdateCursor] = version
			}
		}
	}

	// ensure that the returned versions array is sorted by GVK
	keys := []int{}
	for key, _ := range laterVersions {
		keys = append(keys, key)
	}
	sort.Ints(keys)

	sortedVersions := []versiontypes.AppVersion{}
	for _, key := range keys {
		sortedVersions = append(sortedVersions, laterVersions[key])
	}

	return latestSequenceWithUpdateCursor, sortedVersions, nil
}
