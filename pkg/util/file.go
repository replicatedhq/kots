package util

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"strings"

	"github.com/pkg/errors"
)

func FilesToTGZ(files map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for path, content := range files {
		header := &tar.Header{
			Name: path,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(header); err != nil {
			return nil, errors.Wrapf(err, "failed to write tar header for %s", path)
		}
		_, err := io.Copy(tw, strings.NewReader(content))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to write %s to tar", path)
		}
	}

	if err := tw.Close(); err != nil {
		return nil, errors.Wrap(err, "failed to close tar writer")
	}

	if err := gw.Close(); err != nil {
		return nil, errors.Wrap(err, "failed to close gzip writer")
	}

	return buf.Bytes(), nil
}

func TGZToFiles(tgzBytes []byte) (map[string]string, error) {
	files := make(map[string]string)

	gr, err := gzip.NewReader(bytes.NewReader(tgzBytes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gzip reader")
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if header.Typeflag == tar.TypeReg {
			var contentBuf bytes.Buffer
			if _, err := io.Copy(&contentBuf, tr); err != nil {
				return nil, errors.Wrap(err, "failed to copy tar data")
			}
			files[header.Name] = contentBuf.String()
		}
	}

	return files, nil
}
