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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (w *Worker) GetStateJSONFromSecret(namespace string, name string, key string) ([]byte, error) {
	secret, err := w.K8sClient.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "secret not found")
	}

	return secret.Data[key], nil
}
