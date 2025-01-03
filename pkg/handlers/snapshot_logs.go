package handlers

import (
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	snapshot "github.com/replicatedhq/kots/pkg/kotsadmsnapshot"
	"github.com/replicatedhq/kots/pkg/logger"
	kotssnapshot "github.com/replicatedhq/kots/pkg/snapshot"
	"github.com/replicatedhq/kots/pkg/util"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/client-go/kubernetes"
)

func (h *Handler) DownloadSnapshotLogs(w http.ResponseWriter, r *http.Request) {
	backupName := mux.Vars(r)["backup"]

	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		err = errors.Wrap(err, "failed to get cluster config")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		err = errors.Wrap(err, "failed to create clientset")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	veleroClient, err := k8sutil.GetKubeClient(r.Context())
	if err != nil {
		err = errors.Wrap(err, "failed to create velero client")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	bsl, err := kotssnapshot.FindBackupStoreLocation(r.Context(), clientset, veleroClient, util.PodNamespace)
	if err != nil {
		err = errors.Wrap(err, "failed to find backup store location")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if bsl == nil {
		err = errors.New("no backup store location found")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	veleroNamespace := bsl.Namespace
	gzipReader, err := snapshot.DownloadRequest(r.Context(), veleroNamespace, velerov1.DownloadTargetKindBackupLog, backupName)
	if err != nil {
		err = errors.Wrap(err, "failed to download backup log")
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer gzipReader.Close()

	w.Header().Set("Content-Disposition", "attachment; filename=snapshot.log")
	w.Header().Set("Content-Type", "text/plain")

	w.WriteHeader(http.StatusOK)

	_, err = io.Copy(w, gzipReader)
	if err != nil {
		logger.Error(err)
		return
	}
}
