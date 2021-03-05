package types

import (
	"io"
	"time"

	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/logger"
)

type PushImagesOptions struct {
	Registry       registry.RegistryOptions
	Log            *logger.CLILogger
	ProgressWriter io.Writer
	LogForUI       bool
}

type ImageFile struct {
	Format         string
	Status         string
	Error          string
	FilePath       string
	Layers         map[string]*LayerInfo
	FileSize       int64
	UploadStart    time.Time
	UploadEnd      time.Time
	LayersUploaded int64
}

type LayerInfo struct {
	ID        string
	Size      int64
	UploadEnd time.Time
}
