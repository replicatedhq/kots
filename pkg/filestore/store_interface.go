package filestore

import (
	"context"
	"io"
)

type FileStore interface {
	Init() error
	WaitForReady(ctx context.Context) error
	PutObject(bucket string, key string, body io.ReadSeeker) error
	GetObject(bucket string, key string) (string, error)
}
