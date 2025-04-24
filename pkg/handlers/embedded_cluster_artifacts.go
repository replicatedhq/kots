package handlers

import (
	"archive/tar"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

// GetEmbeddedClusterInfraImages returns the infrastructure images as a .tgz file
func (h *Handler) GetEmbeddedClusterInfraImages(w http.ResponseWriter, r *http.Request) {
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

	// Check if images directory exists
	_, err := os.Stat(imagesDir)
	if os.IsNotExist(err) {
		logger.Error(errors.Wrap(err, "images directory not found"))
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		logger.Error(errors.Wrap(err, "failed to stat images directory"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Find all infra image files
	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to read images directory"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Filter and sort image files
	var infraImages []os.FileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if it's an infra images file
		if !strings.HasSuffix(entry.Name(), "images-amd64.tar") {
			continue
		}

		// Get file info for sorting by modification time
		fileInfo, err := entry.Info()
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to get file info for %s", entry.Name()))
			continue
		}

		fmt.Println("fileInfo", fileInfo.Name(), fileInfo.ModTime())

		infraImages = append(infraImages, fileInfo)
	}

	if len(infraImages) == 0 {
		logger.Error(errors.New("no infrastructure image files found"))
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Sort by modification time, newest first
	sort.Slice(infraImages, func(i, j int) bool {
		return infraImages[i].ModTime().After(infraImages[j].ModTime())
	})

	// Get the newest image (first after sorting)
	newestImage := infraImages[0]

	// Path to newest infra image file
	imagePath := filepath.Join(imagesDir, newestImage.Name())

	// Open image file
	imageFile, err := os.Open(imagePath)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to open image file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer imageFile.Close()

	// Set response headers
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", newestImage.Name()))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", newestImage.Size()))

	// Copy file content to response
	if _, err := io.Copy(w, imageFile); err != nil {
		logger.Error(errors.Wrap(err, "failed to write image file to response"))
		return
	}
}
