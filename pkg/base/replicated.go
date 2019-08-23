package base

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/upstream"
)

func renderReplicated(u *upstream.Upstream, renderOptions *RenderOptions) (*Base, error) {
	baseFiles := []BaseFile{}

	builder := template.Builder{}
	builder.AddCtx(template.StaticCtx{})
	configCtx, err := builder.NewConfigContext(nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create config context")
	}
	builder.AddCtx(configCtx)

	for _, upstreamFile := range u.Files {
		rendered, err := builder.RenderTemplate(upstreamFile.Path, string(upstreamFile.Content))
		if err != nil {
			return nil, errors.Wrap(err, "failed to render template")
		}

		baseFile := BaseFile{
			Path:    upstreamFile.Path,
			Content: []byte(rendered),
		}

		baseFiles = append(baseFiles, baseFile)
	}

	// kustomization :=
	base := Base{
		Files: baseFiles,
	}

	return &base, nil
}
