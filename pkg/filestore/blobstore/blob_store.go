package blobstore

import (
	"context"
	"io"
	"path/filepath"

	"github.com/pkg/errors"
)

var (
	ErrNotImplemented = errors.New("not implemented in ocistore")
)

type BlobStore struct {
}

func (s *BlobStore) Init() error {
	return ErrNotImplemented
}

func (s *BlobStore) WaitForReady(ctx context.Context) error {
	return ErrNotImplemented
}

func (s *BlobStore) PutObject(bucket string, key string, body io.ReadSeeker) error {
	return ErrNotImplemented
}

func (s *BlobStore) GetObject(bucket string, key string) (string, error) {
	return filepath.Join(bucket, key), nil
}
