package handlers

import (
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	snapshot "github.com/replicatedhq/kots/pkg/kotsadmsnapshot"
	"github.com/replicatedhq/kots/pkg/logger"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

func (h *Handler) DownloadSnapshotLogs(w http.ResponseWriter, r *http.Request) {
	backupName := mux.Vars(r)["backup"]

	bsl, err := snapshot.FindBackupStoreLocation()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	veleroNamespace := bsl.Namespace
	gzipReader, err := snapshot.DownloadRequest(veleroNamespace, velerov1.DownloadTargetKindBackupLog, backupName)
	if err != nil {
		err = errors.Wrap(err, "failed to download backup log")
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	defer gzipReader.Close()

	w.Header().Set("Content-Disposition", "attachment; filename=snapshot.log")
	w.Header().Set("Content-Type", "text/plain")

	w.WriteHeader(200)

	_, err = io.Copy(w, gzipReader)
	if err != nil {
		logger.Error(err)
		return
	}
}
