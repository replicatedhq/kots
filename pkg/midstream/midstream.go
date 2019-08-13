package midstream

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/base"
	kustomizetypes "sigs.k8s.io/kustomize/v3/pkg/types"
)

type Midstream struct {
	Kustomization string
}

func CreateMidstream(b *base.Base) (*Midstream, error) {
	kustomization := kustomizetypes.Kustomization{}

	marshalled, err := json.Marshal(kustomization)
	if err != nil {
		return nil, errors.Wrap(err, "marshal midstream")
	}

	m := Midstream{
		Kustomization: string(marshalled),
	}

	return &m, nil
}
