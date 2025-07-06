package handlers

import (
	"archive/tar"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

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

	// Get data directory path from env var
	dataDir := util.EmbeddedClusterDataDir()
	if dataDir == "" {
		logger.Error(errors.New("environment variable EMBEDDED_CLUSTER_DATA_DIR not set"))
		w.WriteHeader(http.StatusInternalServerError)
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

	// Open binary file
	binaryFile, err := os.Open(binaryPath)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to open binary file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer binaryFile.Close()

	// Set response headers for the .tgz file
	filename := fmt.Sprintf("%s.tgz", binaryName)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", "application/gzip")

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
		ModTime: binaryStat.ModTime(),
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

// GetEmbeddedClusterCharts returns the charts in the <data-dir>/charts directory as a .tgz file
func (h *Handler) GetEmbeddedClusterCharts(w http.ResponseWriter, r *http.Request) {
	if !util.IsEmbeddedCluster() {
		logger.Error(errors.New("not an embedded cluster"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get data directory path from env var
	dataDir := util.EmbeddedClusterDataDir()
	if dataDir == "" {
		logger.Error(errors.New("environment variable EMBEDDED_CLUSTER_DATA_DIR not set"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Path to charts directory
	chartsDir := filepath.Join(dataDir, "charts")

	// Create .tgz archive
	archivePath, err := createECChartsArchive(chartsDir)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to create archive"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer os.Remove(archivePath)

	// Open archive file
	archiveFile, err := os.Open(archivePath)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to open archive file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer archiveFile.Close()

	// Get file information
	fileInfo, err := archiveFile.Stat()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get file info"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Disposition", "attachment; filename=ec-charts.tgz")
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Stream the file to the response
	if _, err := io.Copy(w, archiveFile); err != nil {
		logger.Error(errors.Wrap(err, "failed to write archive to response"))
		return
	}
}

// GetEmbeddedClusterK0sImages returns the k0s images file
func (h *Handler) GetEmbeddedClusterK0sImages(w http.ResponseWriter, r *http.Request) {
	if !util.IsEmbeddedCluster() {
		logger.Error(errors.New("not an embedded cluster"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get k0s directory path from env var
	k0sDir := util.EmbeddedClusterK0sDir()
	if k0sDir == "" {
		logger.Error(errors.New("environment variable EMBEDDED_CLUSTER_K0S_DIR not set"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Path to images directory
	imagesDir := filepath.Join(k0sDir, "images")

	// Note: The directory can contain different names:
	// - images-amd64.tar: written on initial installation
	// - ec-images-amd64.tar: written on upgrades (takes precedence)
	// TODO: consolidate the two names to avoid confusion
	imagesPaths := []string{
		filepath.Join(imagesDir, "ec-images-amd64.tar"),
		filepath.Join(imagesDir, "images-amd64.tar"),
	}

	var imagesPath string
	for _, path := range imagesPaths {
		if _, err := os.Stat(path); err == nil {
			imagesPath = path
			break
		}
	}

	if imagesPath == "" {
		logger.Error(errors.New("no k0s images file found"))
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Open images file
	imagesFile, err := os.Open(imagesPath)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to open images file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer imagesFile.Close()

	// Get file information
	fileInfo, err := imagesFile.Stat()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get file info"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Disposition", "attachment; filename=ec-images-amd64.tar")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Copy file content to response
	if _, err := io.Copy(w, imagesFile); err != nil {
		logger.Error(errors.Wrap(err, "failed to write images file to response"))
		return
	}
}

// createECChartsArchive creates a .tgz archive of the given embedded cluster charts directory
func createECChartsArchive(chartsDir string) (string, error) {
	// Create temporary .tgz file
	tmpFile, err := os.CreateTemp("", "ec-charts-*.tgz")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temporary file")
	}
	defer tmpFile.Close()

	// Create gzip writer
	gzipWriter := pgzip.NewWriter(tmpFile)
	defer gzipWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	err = filepath.Walk(chartsDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path for tar header name
		relPath, err := filepath.Rel(chartsDir, filePath)
		if err != nil {
			return errors.Wrapf(err, "failed to get relative path for %s", filePath)
		}

		// Create tar header
		header := &tar.Header{
			Name:    relPath,
			Mode:    int64(info.Mode()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return errors.Wrap(err, "failed to write tar header")
		}

		// Open and copy file contents
		chartFile, err := os.Open(filePath)
		if err != nil {
			return errors.Wrapf(err, "failed to open file %s", filePath)
		}
		defer chartFile.Close()

		// Copy file data into tar writer
		if _, err := io.Copy(tarWriter, chartFile); err != nil {
			return errors.Wrapf(err, "failed to copy file %s to tar archive", filePath)
		}

		return nil
	})

	if err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", errors.Wrap(err, "failed to walk charts directory")
	}

	return tmpFile.Name(), nil
}
