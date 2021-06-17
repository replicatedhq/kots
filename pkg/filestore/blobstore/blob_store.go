package blobstore

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
)

var (
	ErrNotImplemented = errors.New("not implemented in ocistore")
)

type BlobStore struct {
}

func (s *BlobStore) Init() error {
	return nil
}

func (s *BlobStore) WaitForReady(ctx context.Context) error {
	period := 1 * time.Second // TODO: backoff
	for {
		_, err := os.Stat("/kotsadmdata")
		if err == nil {
			logger.Debug("blob store is ready")
			return nil
		}

		select {
		case <-time.After(period):
			continue
		case <-ctx.Done():
			return errors.Wrap(err, "failed to detect a valid object store")
		}
	}
}

func (s *BlobStore) PutObject(bucket string, key string, body io.ReadSeeker) error {
	bucket = "/kotsadmdata"
	outputFilePath := filepath.Join(bucket, key)

	parentPath, _ := filepath.Split(outputFilePath)
	err := os.MkdirAll(parentPath, 0755)
	if err != nil {
		return errors.Wrapf(err, "failed to create directory %q", parentPath)
	}

	fileWriter, err := os.Create(outputFilePath)
	if err != nil {
		return errors.Wrapf(err, "failed to create file %q", outputFilePath)
	}
	defer fileWriter.Close()

	_, err = io.Copy(fileWriter, body)
	if err != nil {
		return errors.Wrapf(err, "failed to write file %q", outputFilePath)
	}

	return nil
}

func (s *BlobStore) GetObject(bucket string, key string) (string, error) {
	bucket = "/kotsadmdata"

	inputFilePath := filepath.Join(bucket, key)
	fileReader, err := os.Open(inputFilePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to open file %q", inputFilePath)
	}
	defer fileReader.Close()

	tmpDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	keyParts := strings.Split(key, string(os.PathSeparator))
	outputFilePath := filepath.Join(tmpDir, keyParts[len(keyParts)-1])

	fileWriter, err := os.Create(outputFilePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create file %q", outputFilePath)
	}
	defer fileWriter.Close()

	_, err = io.Copy(fileWriter, fileReader)
	if err != nil {
		return "", errors.Wrapf(err, "failed to write file %q", outputFilePath)
	}

	return outputFilePath, nil
}
