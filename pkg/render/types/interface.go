package types

import (
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
)

type RenderFileOptions struct {
	KotsKinds        *kotsutil.KotsKinds
	RegistrySettings registrytypes.RegistrySettings
	AppSlug          string
	Sequence         int64
	IsAirgap         bool
	Namespace        string
	InputContent     []byte
}

type RenderDirOptions struct {
	ArchiveDir       string
	App              *apptypes.App
	Downstreams      []downstreamtypes.Downstream
	RegistrySettings registrytypes.RegistrySettings
	Sequence         int64
}

type Renderer interface {
	RenderFile(opts RenderFileOptions) ([]byte, error)
	RenderDir(opts RenderDirOptions) error
}
