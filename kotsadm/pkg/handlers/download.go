package handlers

import (
	"database/sql"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
)

func DownloadApp(w http.ResponseWriter, r *http.Request) {
	if err := requireValidKOTSToken(w, r); err != nil {
		logger.Error(err)
		return
	}

	a, err := store.GetStore().GetAppFromSlug(r.URL.Query().Get("slug"))
	if err != nil {
		logger.Error(err)
		if errors.Is(err, sql.ErrNoRows) {
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

	archivePath, err := version.GetAppVersionArchive(a.ID, a.CurrentSequence)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	if decryptPasswordValues {
		kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archivePath)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
			return
		}

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

	// archiveDir is unarchived, it contains the files
	// let's package that back up for the kots cli
	// because sending 1 file is nice. sending many files, not so nice.
	paths := []string{
		filepath.Join(archivePath, "upstream"),
		filepath.Join(archivePath, "base"),
		filepath.Join(archivePath, "overlays"),
	}

	skippedFilesPath := filepath.Join(archivePath, "skippedFiles")
	if _, err := os.Stat(skippedFilesPath); err == nil {
		paths = append(paths, skippedFilesPath)
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

	w.Header().Set("Content-Disposition", "attachment; filename=archive.tar.gz")
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))

	f, err := os.Open(fileToSend)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	io.Copy(w, f)
	w.WriteHeader(200)
}
