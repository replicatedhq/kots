package types

import (
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	kotsutiltypes "github.com/replicatedhq/kots/pkg/kotsutil/types"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
)

type Renderer interface {
	RenderDir(archiveDir string, kotsKinds *kotsutiltypes.KotsKinds, a *apptypes.App, downstreams []downstreamtypes.Downstream, registrySettings registrytypes.RegistrySettings, sequence int64) error
}
