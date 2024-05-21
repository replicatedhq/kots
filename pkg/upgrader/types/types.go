package types

import (
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
)

type StartOptions struct {
	KOTSVersion      string
	App              *apptypes.App
	AppArchive       string
	RegistrySettings registrytypes.RegistrySettings
}

type ServerParams struct {
	Port string

	AppID       string
	AppSlug     string
	AppSequence int64
	AppIsAirgap bool
	AppLicense  string
	AppArchive  string

	RegistryEndpoint   string
	RegistryUsername   string
	RegistryPassword   string
	RegistryNamespace  string
	RegistryIsReadOnly bool
}
