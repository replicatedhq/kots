package base

import (
	"github.com/replicatedhq/kots/pkg/upstream"
)

func renderReplicated(u *upstream.Upstream, renderOptions *RenderOptions) (*Base, error) {
	baseFiles := []BaseFile{}

	for _, upstreamFile := range u.Files {
		baseFile := BaseFile{
			Path:    upstreamFile.Path,
			Content: upstreamFile.Content,
		}

		baseFiles = append(baseFiles, baseFile)
	}

	// kustomization :=
	base := Base{
		Files: baseFiles,
	}

	return &base, nil
}
