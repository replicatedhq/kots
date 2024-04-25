package helper

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/render"
	rendertypes "github.com/replicatedhq/kots/pkg/render/types"
	"github.com/replicatedhq/kots/pkg/store"
)

// RenderAppFile renders a single file using the current sequence of the provided app, or the overrideSequence (if provided)
// it's here for now to avoid an import cycle between kotsadm/pkg/render and pkg/store
func RenderAppFile(a types.AppType, overrideSequence *int64, inputContent []byte, kotsKinds *kotsutil.KotsKinds, namespace string) ([]byte, error) {
	var sequence int64
	if overrideSequence != nil {
		sequence = *overrideSequence
	} else {
		latestSequence, err := store.GetStore().GetLatestAppSequence(a.GetID(), true)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get latest app sequence")
		}
		sequence = latestSequence
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(a.GetID())
	if err != nil {
		return nil, errors.Wrap(err, "failed to load registry settings")
	}

	return render.RenderFile(rendertypes.RenderFileOptions{
		KotsKinds:        kotsKinds,
		RegistrySettings: registrySettings,
		AppSlug:          a.GetSlug(),
		Sequence:         sequence,
		IsAirgap:         a.GetIsAirgap(),
		Namespace:        namespace,
		InputContent:     []byte(inputContent),
	})
}
