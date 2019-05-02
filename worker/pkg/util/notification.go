package util

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"mime/multipart"
	"strings"

	"github.com/pkg/errors"
)

func FindRendered(file multipart.File) (fileName string, fileContents string, err error) {
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return "", "", errors.Wrap(err, "create gzip reader")
	}

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", "", errors.Wrap(err, "extract tar")
		}

		switch header.Typeflag {
		case tar.TypeReg:
			if strings.HasSuffix(header.Name, "rendered.yaml") {
				fileName = strings.Join(strings.Split(header.Name, "/")[1:], "/")

				data, err := ioutil.ReadAll(tarReader)
				if err != nil {
					return "", "", errors.Wrap(err, "read all")
				}

				fileContents = string(data)
			}
		}
	}

	return fileName, fileContents, nil
}
