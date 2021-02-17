package helper

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/render"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/pkg/kotsutil"
)

// RenderAppFile renders a single file using the current sequence of the provided app, or the overrideSequence (if provided)
// it's here for now to avoid an import cycle between kotsadm/pkg/render and pkg/store
func RenderAppFile(a *types.App, overrideSequence *int64, inputContent []byte, kotsKinds *kotsutil.KotsKinds) ([]byte, error) {
	sequence := a.CurrentSequence
	if overrideSequence != nil {
		sequence = *overrideSequence
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(a.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load registry settings")
	}

	return render.RenderFile(kotsKinds, registrySettings, a.Slug, sequence, a.IsAirgap, inputContent)
}
