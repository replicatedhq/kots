package validators

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/lint/types"
	kurllint "github.com/replicatedhq/kurlkinds/pkg/lint"
)

// KurlLinter wraps the kurlkinds linter
type KurlLinter struct {
	Linter *kurllint.Linter
}

// NewKurlLinter creates a new Kurl linter
func NewKurlLinter() *KurlLinter {
	return &KurlLinter{
		Linter: kurllint.New(),
	}
}

// ValidateKurlInstaller searches installer yamls for errors or misconfigurations
func (kurlLinter *KurlLinter) ValidateKurlInstaller(specFiles types.SpecFiles) ([]types.LintExpression, error) {
	separated, err := specFiles.Separate()
	if err != nil {
		return nil, errors.Wrap(err, "error separating spec files")
	}

	var expressions []types.LintExpression
	for _, file := range separated {
		if !file.IsYAML() {
			continue
		}

		output, err := kurlLinter.Linter.ValidateMarshaledYAML(context.Background(), file.Content)
		if err != nil {
			if err != kurllint.ErrNotInstaller {
				return nil, errors.Wrap(err, "unable to lint installer")
			}
			continue
		}

		for _, out := range output {
			expressions = append(
				expressions, types.LintExpression{
					Rule:    fmt.Sprintf("kubernetes-installer-%s", out.Type),
					Type:    "error",
					Path:    file.Path,
					Message: out.Message,
				},
			)
		}
	}
	return expressions, nil
}
