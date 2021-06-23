package filestore

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

const (
	ArchivesDir = "/kotsadmdata/archives"
)

func Init() error {
	err := os.MkdirAll(ArchivesDir, 0755)
	if err != nil {
		return errors.Wrapf(err, "failed to create archives directory")
	}
	return nil
}

func WriteArchive(outputPath string, body io.ReadSeeker) error {
	return writeFile(filepath.Join(ArchivesDir, outputPath), body)
}

func writeFile(outputPath string, body io.ReadSeeker) error {
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

func ReadArchive(path string) (string, error) {
	return readFile(filepath.Join(ArchivesDir, path))
}

// readFile creates a new copy of the file under /tmp and returns the path for it.
// the caller is responsible for cleaning up.
// this is so that the original files are not removed by the caller on cleanup by mistake.
func readFile(path string) (string, error) {
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
