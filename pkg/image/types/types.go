package types

import (
	"io"

	"github.com/containers/image/v5/types"
)

type RegistryAuth struct {
	Username string
	Password string
}

type ImageInfo struct {
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
