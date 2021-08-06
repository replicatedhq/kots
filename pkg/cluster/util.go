package cluster

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func extractOneFileFromArchiveStreamToDir(filename string, r io.ReadCloser, dest string) error {
	uncompressedStream, err := gzip.NewReader(r)
	if err != nil {
		return errors.Wrap(err, "create gzip reader")
	}

	tarReader := tar.NewReader(uncompressedStream)

	for true {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return errors.Wrap(err, "read next")
		}

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			if !strings.HasSuffix(header.Name, filename) {
				continue
			}
			outFile, err := os.Create(filepath.Join(dest, filename))
			if err != nil {
				return errors.Wrap(err, "create")
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return errors.Wrap(err, "copy")
			}
			if err := os.Chmod(filepath.Join(dest, filename), fs.FileMode(header.Mode)); err != nil {
				return errors.Wrap(err, "chmod")
			}

			outFile.Close()

		default:
			return errors.New("unknown type")
		}

	}

	return nil
}
