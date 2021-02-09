package types

import (
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	registrytypes "github.com/replicatedhq/kots/kotsadm/pkg/registry/types"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
)

type Renderer interface {
	RenderFile(kotsKinds *kotsutil.KotsKinds, registrySettings *registrytypes.RegistrySettings, appSlug string, sequence int64, isAirgap bool, inputContent []byte) ([]byte, error)
	RenderDir(archiveDir string, a *apptypes.App, downstreams []downstreamtypes.Downstream, registrySettings *registrytypes.RegistrySettings) error
}
