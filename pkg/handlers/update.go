package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	kotspull "github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kotsadm/pkg/app"
	"github.com/replicatedhq/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kotsadm/pkg/license"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/session"
)

type AppUpdateCheckRequest struct {
}

type AppUpdateCheckResponse struct {
	AvailableUpdates int64 `json:"availableUpdates"`
}

func AppUpdateCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		w.WriteHeader(401)
		return
	}

	foundApp, err := app.GetFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	currentStatus, err := app.GetTaskStatus("update-download")
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	appUpdateCheckResponse := AppUpdateCheckResponse{
		AvailableUpdates: 0,
	}

	if currentStatus == "running" {
		logger.Debug("update-download is already running, not starting a new one")
		JSON(w, 200, appUpdateCheckResponse)
		return
	}

	if err := app.ClearTaskStatus("update-download"); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	// sync license, this method is only called when online
	_, err = license.Sync(foundApp, "")
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	// download the app
	archiveDir, err := app.GetAppVersionArchive(foundApp.ID, foundApp.CurrentSequence)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	// we need a few objects from the app to check for updates
	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	getUpdatesOptions := kotspull.GetUpdatesOptions{
		LicenseFile:    filepath.Join(archiveDir, "upstream", "userdata", "license.yaml"),
		CurrentCursor:  kotsKinds.Installation.Spec.UpdateCursor,
		CurrentChannel: kotsKinds.Installation.Spec.ChannelName,
		Silent:         false,
	}

	updates, err := kotspull.GetUpdates(fmt.Sprintf("replicated://%s", kotsKinds.License.Spec.AppSlug), getUpdatesOptions)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	// update last updated at time
	t := app.LastUpdateAtTime(foundApp.ID)
	if t != nil {
		logger.Error(t)
		w.WriteHeader(500)
		return
	}

	// if there are updates, go routine it
	if len(updates) == 0 {
		JSON(w, 200, appUpdateCheckResponse)
		return
	}

	appUpdateCheckResponse.AvailableUpdates = int64(len(updates))

	go func() {
		defer os.RemoveAll(archiveDir)
		for _, update := range updates {
			// the latest version is in archive dir
			if err := app.DownloadUpdate(foundApp, archiveDir, update.Cursor); err != nil {
				logger.Error(err)
			}

		}

	}()

	JSON(w, 200, appUpdateCheckResponse)
}
