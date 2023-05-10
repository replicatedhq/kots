package archives

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// todo, figure out why this doesn't use the mholt tgz archiver that we
// use elsewhere in kots
func ExtractTGZArchiveFromFile(tgzFile string, destDir string) error {
	fileReader, err := os.Open(tgzFile)
	if err != nil {
		return errors.Wrap(err, "failed to open tgz file")
	}
	defer fileReader.Close()

	err = ExtractTGZArchiveFromReader(fileReader, destDir)
	if err != nil {
		return errors.Wrap(err, "failed to extract archive")
	}

	return nil
}

func GetFileFromAirgap(fileToGet string, archive string) ([]byte, error) {
	fileReader, err := os.Open(archive)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}
	defer fileReader.Close()

	gzipReader, err := gzip.NewReader(fileReader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get new gzip reader")
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	var fileData []byte
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "failed to get read archive")
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}
		if header.Name != fileToGet {
			continue
		}

		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(tarReader)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read file from tar archive")
		}

		fileData = buf.Bytes()
		break
	}

	if fileData == nil {
		return nil, errors.New("file not found in archive")
	}

	return fileData, nil
}

func ExtractTGZArchiveFromReader(tgzReader io.Reader, destDir string) error {
	gzReader, err := gzip.NewReader(tgzReader)
	if err != nil {
		return errors.Wrap(err, "failed to get new gzip reader")
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "failed to read tar data")
		}

		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		err = func() error {
			fileName := filepath.Join(destDir, hdr.Name)

			filePath, _ := filepath.Split(fileName)
			err := os.MkdirAll(filePath, 0755)
			if err != nil {
				return errors.Wrapf(err, "failed to create directory %q", filePath)
			}

			fileWriter, err := os.Create(fileName)
			if err != nil {
				return errors.Wrapf(err, "failed to create file %q", hdr.Name)
			}

			defer fileWriter.Close()

			_, err = io.Copy(fileWriter, tarReader)
			if err != nil {
				return errors.Wrapf(err, "failed to write file %q", hdr.Name)
			}

			return nil
		}()
		if err != nil {
			return err
		}
	}

	return nil
}

func IsTGZ(b []byte) bool {
	r := bytes.NewReader(b)

	gzipReader, err := gzip.NewReader(r)
	if err != nil {
		return false
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	// try to read the first file header from the tar archive
	_, err = tarReader.Next()
	if err != nil {
		if err == io.EOF {
			return false
		}
		return false
	}

	return true
}
