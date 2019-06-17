package watchworker

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"mime/multipart"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
)

func (w *Worker) GetStateJSONFromArchive(logger log.Logger, file multipart.File) ([]byte, error) {
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, errors.Wrap(err, "create gzip reader")
	}

	tarReader := tar.NewReader(gzipReader)
	var data []byte

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "extract tar")
		}

		switch header.Typeflag {
		case tar.TypeReg:
			if strings.HasSuffix(header.Name, "/state.json") {
				content, err := ioutil.ReadAll(tarReader)
				if err != nil {
					level.Error(logger).Log("event", "readfile", "err", err)
				}

				data = content
			}
		}
	}

	return data, nil
}
