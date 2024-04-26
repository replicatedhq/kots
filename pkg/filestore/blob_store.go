package filestore

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

var (
	ArchivesDir = "/kotsadmdata/archives"
)

type BlobStore struct {
}

func (s *BlobStore) Init() error {
	err := os.MkdirAll(ArchivesDir, 0755)
	if err != nil {
		return errors.Wrapf(err, "failed to create archives directory")
	}
	return nil
}

func (s *BlobStore) WaitForReady(ctx context.Context) error {
	// there's no waiting, it's either there or it's not. the pod won't come up if the volume didn't mount.
	return nil
}

func (s *BlobStore) WriteArchive(outputPath string, body io.ReadSeeker) error {
	return s.writeFile(filepath.Join(ArchivesDir, outputPath), body)
}

func (s *BlobStore) writeFile(outputPath string, body io.ReadSeeker) error {
	parentPath, _ := filepath.Split(outputPath)
	err := os.MkdirAll(parentPath, 0755)
	if err != nil {
		return errors.Wrapf(err, "failed to create directory %q", parentPath)
	}

	fileWriter, err := os.Create(outputPath)
	if err != nil {
		return errors.Wrapf(err, "failed to create file %q", outputPath)
	}
	defer fileWriter.Close()

	_, err = io.Copy(fileWriter, body)
	if err != nil {
		return errors.Wrapf(err, "failed to write file %q", outputPath)
	}

	return nil
}

func (s *BlobStore) ReadArchive(path string) (string, error) {
	return s.readFile(filepath.Join(ArchivesDir, path))
}

// readFile creates a new copy of the file under /tmp and returns the path for it.
// the caller is responsible for cleaning up.
// this is so that the original files are not removed by the caller on cleanup by mistake.
func (s *BlobStore) readFile(path string) (string, error) {
	fileReader, err := os.Open(path)
	if err != nil {
		return "", errors.Wrapf(err, "failed to open file %q", path)
	}
	defer fileReader.Close()

	tmpDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	pathParts := strings.Split(path, string(os.PathSeparator))
	outputPath := filepath.Join(tmpDir, pathParts[len(pathParts)-1])

	fileWriter, err := os.Create(outputPath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create file %q", outputPath)
	}
	defer fileWriter.Close()

	_, err = io.Copy(fileWriter, fileReader)
	if err != nil {
		return "", errors.Wrapf(err, "failed to write file %q", outputPath)
	}

	return outputPath, nil
}

func (s *BlobStore) DeleteArchive(path string) error {
	return s.deleteFile(filepath.Join(ArchivesDir, path))
}

func (s *BlobStore) deleteFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	err := os.RemoveAll(path)
	if err != nil {
		return errors.Wrapf(err, "failed to remove file %q", path)
	}
	return nil
}
