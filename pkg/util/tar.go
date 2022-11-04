package util

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

func ExtractTGZArchive(tgzFile string, destDir string) error {
	fileReader, err := os.Open(tgzFile)
	if err != nil {
		return errors.Wrap(err, "failed to open tgz file")
	}

	gzReader, err := gzip.NewReader(fileReader)
	if err != nil {
		return errors.Wrap(err, "failed to create gzip reader")
	}

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

func GetFileFromTGZArchive(archive *bytes.Buffer, fileName string) (*bytes.Buffer, error) {
	gzReader, err := gzip.NewReader(archive)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gzip reader")
	}

	tarReader := tar.NewReader(gzReader)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "failed to read tar data")
		}

		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		match, err := filepath.Match(fileName, hdr.Name)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check filename match")
		}

		if !match {
			_, err = io.Copy(io.Discard, tarReader)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to discard file %q", hdr.Name)
			}
		} else {
			buf := bytes.NewBuffer(nil)
			_, err = io.Copy(buf, tarReader)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to copy file %q", hdr.Name)
			}
			return bytes.NewBuffer(buf.Bytes()), nil
		}
	}

	return nil, errors.Errorf("file %s not found in archive", fileName)
}

// TODO NOW: write a unit test
func ReplaceFileInTGZArchive(archive *bytes.Buffer, fileName string, newContents []byte) (*bytes.Buffer, error) {
	if len(archive.Bytes()) == 0 {
		return bytes.NewBuffer(nil), nil
	}

	gzReader, err := gzip.NewReader(archive)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gzip reader")
	}
	tarReader := tar.NewReader(gzReader)

	buf := bytes.NewBuffer(nil)
	gzWriter := gzip.NewWriter(buf)
	defer gzWriter.Close()
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "failed to read tar data")
		}

		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		match, err := filepath.Match(fileName, hdr.Name)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check filename match")
		}

		var contents []byte
		var size int64
		var modTime time.Time

		if match {
			_, err = io.Copy(io.Discard, tarReader)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to discard file %q", hdr.Name)
			}
			contents = newContents
			size = int64(len(newContents))
			modTime = time.Now()
		} else {
			buf := bytes.NewBuffer(nil)
			_, err = io.Copy(buf, tarReader)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to copy file %q", hdr.Name)
			}
			contents = buf.Bytes()
			size = hdr.Size
			modTime = hdr.ModTime
		}

		newHdr := &tar.Header{
			Name:    hdr.Name,
			Mode:    hdr.Mode,
			Size:    size,
			ModTime: modTime,
		}
		if err := tarWriter.WriteHeader(newHdr); err != nil {
			return nil, errors.Wrap(err, "failed to write tar header")
		}
		if _, err := tarWriter.Write(contents); err != nil {
			return nil, errors.Wrap(err, "failed to write tar contents")
		}
	}

	return buf, nil
}
