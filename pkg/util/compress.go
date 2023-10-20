package util

import (
	"bytes"
	"compress/gzip"
	"io"

	"github.com/pkg/errors"
)

func GzipData(input []byte) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)

	_, err := gw.Write(input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write to gzip writer")
	}

	err = gw.Close()
	if err != nil {
		return nil, errors.Wrap(err, "failed to close gzip writer")
	}

	return buf.Bytes(), nil
}

func GunzipData(input []byte) ([]byte, error) {
	r := bytes.NewReader(input)
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gzip reader")
	}
	defer gr.Close()

	decompressedData, err := io.ReadAll(gr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read from gzip reader")
	}

	return decompressedData, nil
}
