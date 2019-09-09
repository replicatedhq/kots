package util

import (
	"io"
	"fmt"
	"os"
	"bytes"
	"crypto/md5"
	"archive/tar"
	"compress/gzip"

	"github.com/pkg/errors"
)

type CompareOptions struct {
	IgnoreFilesInActual []string
}

// CompareTars will return a bool if the tars files match, without
// comparing timestmaps and more. t
func CompareTars(expected []byte, actual []byte, compareOptions CompareOptions) (bool, error) {
	expectedFiles, err := parseTar(expected)
	if err != nil {
		return false, errors.Wrap(err, "failed to parse epxected tar")
	}

	actualFiles, err := parseTar(actual)
	if err != nil {
		return false, errors.Wrap(err, "failed to parse actual tar")
	}

	unexpectedFiles := []string{}
	for actualFilename, _ := range actualFiles {
		for _, ignoredFile := range compareOptions.IgnoreFilesInActual {
			if actualFilename == ignoredFile {
				goto NextFile
			}
		}
		if _, ok := expectedFiles[actualFilename]; !ok {
			unexpectedFiles = append(unexpectedFiles, actualFilename)
		}

		NextFile:
	}

	if len(unexpectedFiles) > 0 {
		return false, errors.Errorf("unexpected files: %#v\n", unexpectedFiles)
	}

	return  true, nil
}

func parseTar(in []byte) (map[string]string, error) {
	files := map[string]string{}

	byteReader := bytes.NewReader(in)
	gzf, err := gzip.NewReader(byteReader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get new gzip reader")
	}

	tarReader := tar.NewReader(gzf)

	i := 0
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		name := header.Name

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(tarReader)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read file from tar archive")
			}

			files[name] = fmt.Sprintf("%x", md5.Sum(buf.Bytes()))
		}

		i++
	}

	return files, nil
}