package filestore

import (
	"context"
	"io"
)

type FileStore interface {
	Init() error
	WaitForReady(ctx context.Context) error
	WriteArchive(outputPath string, body io.ReadSeeker) error
	ReadArchive(path string) (string, error)
}
