package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/marccampbell/yaml-toolbox/pkg/splitter"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/pkg/kotsutil"
)

type GetAppRenderedContentsResponse struct {
	Files map[string]string `json:"files"`
}

type GetAppRenderedContentsErrorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) GetAppRenderedContents(w http.ResponseWriter, r *http.Request) {
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

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archivePath)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	// pick the first downstream found
	// which will likely be "this-cluster"
	children, err := ioutil.ReadDir(filepath.Join(archivePath, "overlays", "downstreams"))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	downstreamName := ""
	for _, child := range children {
		if child.IsDir() && child.Name() != "." && child.Name() != ".." {
			downstreamName = child.Name()
		}
	}

	kustomizeBuildTarget := ""

	if downstreamName == "" {
		kustomizeBuildTarget = filepath.Join(archivePath, "overlays", "midstream")
	} else {
		kustomizeBuildTarget = filepath.Join(archivePath, "overlays", "downstreams", downstreamName)
	}

	archiveOutput, err := exec.Command(fmt.Sprintf("kustomize%s", kotsKinds.KustomizeVersion()), "build", kustomizeBuildTarget).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("kustomize stderr: %q", string(ee.Stderr))
			logger.Error(err)

			JSON(w, 500, GetAppRenderedContentsErrorResponse{
				Error: fmt.Sprintf("Failed to build release: %v", err),
			})
			return

		}
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	archiveFiles, err := splitter.SplitYAML(archiveOutput)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	// base64 decode these
	decodedArchiveFiles := map[string]string{}
	for filename, b := range archiveFiles {
		decodedArchiveFiles[filename] = string(b)
	}
	JSON(w, 200, GetAppRenderedContentsResponse{
		Files: decodedArchiveFiles,
	})
}
