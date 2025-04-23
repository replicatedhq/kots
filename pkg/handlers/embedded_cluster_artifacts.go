package handlers

import (
	"archive/tar"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/klauspost/pgzip"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
)

// GetEmbeddedClusterBinary returns the embedded cluster binary as a .tgz file
func (h *Handler) GetEmbeddedClusterBinary(w http.ResponseWriter, r *http.Request) {
	if !util.IsEmbeddedCluster() {
		logger.Error(errors.New("not an embedded cluster"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get kubeclient
	kbClient, err := h.GetKubeClient(r.Context())
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get kubeclient"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Get current installation
	installation, err := embeddedcluster.GetCurrentInstallation(r.Context(), kbClient)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get current installation"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Get data directory path from runtime config
	dataDir := ""
	if installation.Spec.RuntimeConfig != nil {
		dataDir = installation.Spec.RuntimeConfig.DataDir
	}
	if dataDir == "" {
		logger.Error(errors.New("data directory not found in runtime config"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Get binary name from installation
	binaryName := installation.Spec.BinaryName
	if binaryName == "" {
		logger.Error(errors.New("binary name not found in installation"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Path to EC binary
	binaryPath := filepath.Join(dataDir, "bin", binaryName)

	// Check if binary exists
	binaryStat, err := os.Stat(binaryPath)
	if os.IsNotExist(err) {
		logger.Error(errors.Wrap(err, "binary file not found"))
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		logger.Error(errors.Wrap(err, "failed to stat binary file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Set response headers for the .tgz file
	filename := fmt.Sprintf("%s.tgz", binaryName)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", "application/gzip")

	// Open binary file
	binaryFile, err := os.Open(binaryPath)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to open binary file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer binaryFile.Close()

	// Create pgzip writer
	gzipWriter := pgzip.NewWriter(w)
	defer gzipWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Add binary file to tar archive
	header := &tar.Header{
		Name:    binaryName,
		Mode:    0755, // Executable permission
		Size:    binaryStat.Size(),
		ModTime: time.Now(),
		Format:  tar.FormatGNU,
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		logger.Error(errors.Wrap(err, "failed to write tar header"))
		return
	}

	// Copy binary content to tar archive
	if _, err := io.Copy(tarWriter, binaryFile); err != nil {
		logger.Error(errors.Wrap(err, "failed to write binary to tar archive"))
		return
	}
}
