package types

import (
	"io"
	"time"

	registrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	imagetypes "github.com/replicatedhq/kots/pkg/image/types"
	"github.com/replicatedhq/kots/pkg/logger"
)

type PushImagesOptions struct {
	Registry       registrytypes.RegistryOptions
	KotsadmTag     string
	Log            *logger.CLILogger
	ProgressWriter io.Writer
	LogForUI       bool
}

type PushAppImageOptions struct {
	ImageID          string
	ImageInfo        *ImageInfo
	Log              *logger.CLILogger
	LogForUI         bool
	ReportWriter     io.Writer
	CopyImageOptions imagetypes.CopyImageOptions
}

type ImageInfo struct {
	Format      string
	Status      string
	Error       string
	Layers      map[string]*LayerInfo
	UploadStart time.Time
	UploadEnd   time.Time
}

type LayerInfo struct {
	ID          string
	Size        int64
	UploadStart time.Time
	UploadEnd   time.Time
}
