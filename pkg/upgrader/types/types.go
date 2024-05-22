package types

import (
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
)

type StartOptions struct {
	KOTSVersion      string
	App              *apptypes.App
	BaseArchive      string
	BaseSequence     int64
	NextSequence     int64
	UpdateCursor     string
	RegistrySettings registrytypes.RegistrySettings
}

type ServerParams struct {
	Port string

	AppID       string
	AppSlug     string
	AppIsAirgap bool
	AppIsGitOps bool
	AppLicense  string

	BaseArchive  string
	BaseSequence int64
	NextSequence int64

	UpdateCursor string

	RegistryEndpoint   string
	RegistryUsername   string
	RegistryPassword   string
	RegistryNamespace  string
	RegistryIsReadOnly bool
}
