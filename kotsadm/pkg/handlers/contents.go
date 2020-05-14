package handlers

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
)

type GetAppContentsResponse struct {
	Files map[string][]byte `json:"files"`
}

func GetAppContents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	appSlug := mux.Vars(r)["appSlug"]
	sequence, err := strconv.Atoi(mux.Vars(r)["sequence"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	a, err := app.GetFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	archivePath, err := version.GetAppVersionArchive(a.ID, int64(sequence))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	defer os.RemoveAll(archivePath)

	// walk the parth, adding all to the files map
	// base64 decode these
	archiveFiles := map[string][]byte{}

	err = filepath.Walk(archivePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		archiveFiles[strings.TrimPrefix(path, archivePath)] = contents
		return nil
	})
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	getAppContentsResponse := GetAppContentsResponse{
		Files: archiveFiles,
	}

	JSON(w, 200, getAppContentsResponse)
}
