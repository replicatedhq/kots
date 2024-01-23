package types

import (
	"io"
	"time"

	"github.com/containers/image/v5/types"
	dockerregistrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/logger"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
)

type ProcessImageOptions struct {
	AppSlug          string
	Namespace        string
	RewriteImages    bool
	CopyImages       bool
	RegistrySettings registrytypes.RegistrySettings
	RootDir          string
	IsAirgap         bool
	AirgapRoot       string
	AirgapBundle     string
	CreateAppDir     bool
	ReportWriter     io.Writer
}

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
	SrcAuth           RegistryAuth
	DestAuth          RegistryAuth
	CopyAll           bool
	SrcDisableV1Ping  bool
	SrcSkipTLSVerify  bool
	DestDisableV1Ping bool
	DestSkipTLSVerify bool
	ReportWriter      io.Writer
}

type PushImagesOptions struct {
	Registry       dockerregistrytypes.RegistryOptions
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
