package types

import (
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
)

type Renderer interface {
	RenderFile(kotsKinds *kotsutil.KotsKinds, registrySettings registrytypes.RegistrySettings, appSlug string, sequence int64, isAirgap bool, namespace string, inputContent []byte) ([]byte, error)
	RenderDir(archiveDir string, a *apptypes.App, downstreams []downstreamtypes.Downstream, registrySettings registrytypes.RegistrySettings, createNewVersion bool) error
}
