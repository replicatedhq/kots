package archives

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func ExtractAppMetaFromAirgapBundle(airgapBundle string) (string, error) {
	destDir, err := os.MkdirTemp("", "kotsadm-app-meta-")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}
	metaFiles := []string{
		"airgap.yaml",
		"app.tar.gz",
	}
	for _, fileName := range metaFiles {
		content, err := GetFileContentFromTGZArchive(fileName, airgapBundle)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get %s from bundle", fileName)
		}
		if err := os.WriteFile(filepath.Join(destDir, fileName), []byte(content), 0644); err != nil {
			return "", errors.Wrapf(err, "failed to write %s", fileName)
		}
	}
	return destDir, nil
}

func FilterAirgapBundle(airgapBundle string, filesToKeep []string) (string, error) {
	f, err := os.CreateTemp("", "kots-airgap")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp file")
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	fileFilter := make(map[string]bool)
	for _, file := range filesToKeep {
		fileFilter[file] = true
	}

	fileReader, err := os.Open(airgapBundle)
	if err != nil {
		return "", errors.Wrap(err, "failed to open airgap bundle")
	}
	defer fileReader.Close()

	gzipReader, err := gzip.NewReader(fileReader)
	if err != nil {
		return "", errors.Wrap(err, "failed to get new gzip reader")
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", errors.Wrap(err, "failed to get read archive")
		}

		if _, ok := fileFilter[header.Name]; !ok {
			continue
		}

		if err := tw.WriteHeader(header); err != nil {
			return "", errors.Wrapf(err, "failed to write tar header for %s", header.Name)
		}
		_, err = io.Copy(tw, tarReader)
		if err != nil {
			return "", errors.Wrapf(err, "failed to write %s to tar", header.Name)
		}
	}

	if err := tw.Close(); err != nil {
		return "", errors.Wrap(err, "failed to close tar writer")
	}

	if err := gw.Close(); err != nil {
		return "", errors.Wrap(err, "failed to close gzip writer")
	}

	return f.Name(), nil
}
