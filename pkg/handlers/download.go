package handlers

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/mholt/archiver/v3"
	"github.com/pkg/errors"
	versiontypes "github.com/replicatedhq/kots/pkg/api/version/types"
	upstream "github.com/replicatedhq/kots/pkg/kotsadmupstream"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
)

// NOTE: this uses special kots token authorization
func (h *Handler) DownloadApp(w http.ResponseWriter, r *http.Request) {
	if err := requireValidKOTSToken(w, r); err != nil {
		logger.Error(err)
		return
	}

	a, err := store.GetStore().GetAppFromSlug(r.URL.Query().Get("slug"))
	if err != nil {
		logger.Error(err)
		if store.GetStore().IsNotFound(err) {
			w.WriteHeader(404)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	decryptPasswordValues := false
	if r.URL.Query().Get("decryptPasswordValues") != "" {
		decryptPasswordValues, err = strconv.ParseBool(r.URL.Query().Get("decryptPasswordValues"))
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
			return
		}
	}

	latestSequence, err := store.GetStore().GetLatestAppSequence(a.ID, true)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	archivePath, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	defer os.RemoveAll(archivePath)

	err = store.GetStore().GetAppVersionArchive(a.ID, latestSequence, archivePath)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	if decryptPasswordValues {
		kotsKinds, err := kotsutil.LoadKotsKindsFromPath(filepath.Join(archivePath, "upstream"))
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
			return
		}

		if kotsKinds.ConfigValues != nil {
			if err := kotsKinds.DecryptConfigValues(); err != nil {
				logger.Error(err)
				w.WriteHeader(500)
				return
			}

			updated, err := kotsKinds.Marshal("kots.io", "v1beta1", "ConfigValues")
			if err != nil {
				logger.Error(err)
				w.WriteHeader(500)
				return
			}

			if err := ioutil.WriteFile(filepath.Join(archivePath, "upstream", "userdata", "config.yaml"), []byte(updated), 0644); err != nil {
				logger.Error(err)
				w.WriteHeader(500)
				return
			}
		}
	}

	// archiveDir is unarchived, it contains the files
	// let's package that back up for the kots cli
	// because sending 1 file is nice. sending many files, not so nice.
	paths := []string{
		filepath.Join(archivePath, "upstream"),
		filepath.Join(archivePath, "base"),
		filepath.Join(archivePath, "overlays"),
	}

	renderedPath := filepath.Join(archivePath, "rendered")
	if _, err := os.Stat(renderedPath); err == nil {
		paths = append(paths, renderedPath)
	}

	skippedFilesPath := filepath.Join(archivePath, "skippedFiles")
	if _, err := os.Stat(skippedFilesPath); err == nil {
		paths = append(paths, skippedFilesPath)
	}

	kotsKindsPath := filepath.Join(archivePath, "kotsKinds")
	if _, err := os.Stat(kotsKindsPath); err == nil {
		paths = append(paths, kotsKindsPath)
	}

	helmPath := filepath.Join(archivePath, "helm")
	if _, err := os.Stat(helmPath); err == nil {
		paths = append(paths, helmPath)
	}

	tmpDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	defer os.RemoveAll(tmpDir)
	fileToSend := filepath.Join(tmpDir, "archive.tar.gz")

	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Archive(paths, fileToSend); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	fi, err := os.Stat(fileToSend)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	f, err := os.Open(fileToSend)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=archive.tar.gz")
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	w.WriteHeader(200)

	_, err = io.Copy(w, f)
	if err != nil {
		logger.Error(err)
	}
}

type DownloadAppVersionResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) DownloadAppVersion(w http.ResponseWriter, r *http.Request) {
	downloadUpstreamVersionResponse := DownloadAppVersionResponse{
		Success: false,
	}

	appSlug := mux.Vars(r)["appSlug"]

	a, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get app for slug %s", appSlug)
		logger.Error(errors.Wrap(err, errMsg))
		downloadUpstreamVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, downloadUpstreamVersionResponse)
		return
	}

	sequence, err := strconv.Atoi(mux.Vars(r)["sequence"])
	if err != nil {
		errMsg := "failed to parse sequence number"
		logger.Error(errors.Wrap(err, errMsg))
		downloadUpstreamVersionResponse.Error = errMsg
		JSON(w, http.StatusBadRequest, downloadUpstreamVersionResponse)
		return
	}

	skipPreflights, _ := strconv.ParseBool(r.URL.Query().Get("skipPreflights"))
	skipCompatibilityCheck, _ := strconv.ParseBool(r.URL.Query().Get("skipCompatibilityCheck"))
	wait, _ := strconv.ParseBool(r.URL.Query().Get("wait"))

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		errMsg := "failed to list downstreams for app"
		logger.Error(errors.Wrap(err, errMsg))
		downloadUpstreamVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, downloadUpstreamVersionResponse)
		return
	} else if len(downstreams) == 0 {
		errMsg := fmt.Sprintf("no downstreams for app %s", appSlug)
		logger.Error(errors.New(errMsg))
		downloadUpstreamVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, downloadUpstreamVersionResponse)
		return
	}

	version, err := store.GetStore().GetAppVersion(a.ID, int64(sequence))
	if err != nil {
		if store.GetStore().IsNotFound(err) {
			errMsg := fmt.Sprintf("version for sequence %d not found", sequence)
			logger.Error(errors.New(errMsg))
			downloadUpstreamVersionResponse.Error = errMsg
			JSON(w, http.StatusNotFound, downloadUpstreamVersionResponse)
			return
		}
		errMsg := fmt.Sprintf("failed to get app version %d", sequence)
		logger.Error(errors.Wrap(err, errMsg))
		downloadUpstreamVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, downloadUpstreamVersionResponse)
		return
	}

	status, err := store.GetStore().GetStatusForVersion(a.ID, downstreams[0].ClusterID, version.Sequence)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get status for version %d", version.Sequence)
		logger.Error(errors.Wrap(err, errMsg))
		downloadUpstreamVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, downloadUpstreamVersionResponse)
		return
	}
	if status != storetypes.VersionPendingDownload {
		errMsg := fmt.Sprintf("not downloading version %d because it's %s", version.Sequence, status)
		logger.Error(errors.New(errMsg))
		downloadUpstreamVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, downloadUpstreamVersionResponse)
		return
	}

	downloadFn := func(appID string, version *versiontypes.AppVersion, skipPreflights bool, skipCompatibilityCheck bool) error {
		appSequence := version.Sequence
		update := upstreamtypes.Update{
			ChannelID:    version.KOTSKinds.Installation.Spec.ChannelID,
			ChannelName:  version.KOTSKinds.Installation.Spec.ChannelName,
			Cursor:       version.KOTSKinds.Installation.Spec.UpdateCursor,
			VersionLabel: version.KOTSKinds.Installation.Spec.VersionLabel,
			IsRequired:   version.KOTSKinds.Installation.Spec.IsRequired,
			AppSequence:  &appSequence,
		}
		_, err := upstream.DownloadUpdate(appID, update, skipPreflights, skipCompatibilityCheck)
		if err != nil {
			return errors.Wrapf(err, "failed to download update %s", update.VersionLabel)
		}
		return nil
	}

	if wait {
		if err := downloadFn(a.ID, version, skipPreflights, skipCompatibilityCheck); err != nil {
			cause := errors.Cause(err)
			if _, ok := cause.(util.ActionableError); ok {
				downloadUpstreamVersionResponse.Error = cause.Error()
			} else {
				downloadUpstreamVersionResponse.Error = fmt.Sprintf("failed to get app version %d", sequence)
			}
			logger.Error(errors.Wrap(err, "failed synchronously"))
			JSON(w, http.StatusInternalServerError, downloadUpstreamVersionResponse)
			return
		}
	} else {
		go func() {
			if err := downloadFn(a.ID, version, skipPreflights, skipCompatibilityCheck); err != nil {
				logger.Error(errors.Wrap(err, "failed asynchronously"))
			}
		}()
	}

	downloadUpstreamVersionResponse.Success = true

	JSON(w, http.StatusOK, downloadUpstreamVersionResponse)
}
