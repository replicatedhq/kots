package types

import (
	"io"
	"time"

	"github.com/containers/image/v5/types"
	registrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/logger"
)

type RegistryAuth struct {
	Username string
	Password string
}

type InstallationImageInfo struct {
	IsPrivate bool
}

type CopyImageOptions struct {
	SrcRef            types.ImageReference
	DestRef           types.ImageReference
	DestAuth          RegistryAuth
	CopyAll           bool
	SkipSrcTLSVerify  bool
	SkipDestTLSVerify bool
	ReportWriter      io.Writer
}

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
	CopyImageOptions CopyImageOptions
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
