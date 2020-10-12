package handlers

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
)

type GetAppContentsResponse struct {
	Files map[string][]byte `json:"files"`
}

func GetAppContents(w http.ResponseWriter, r *http.Request) {
	appSlug := mux.Vars(r)["appSlug"]
	sequence, err := strconv.Atoi(mux.Vars(r)["sequence"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	a, err := store.GetStore().GetAppFromSlug(appSlug)
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

	err = store.GetStore().GetAppVersionArchive(a.ID, int64(sequence), archivePath)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

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
