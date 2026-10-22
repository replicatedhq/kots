package kotsutil

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	troubleshootloader "github.com/replicatedhq/troubleshoot/pkg/loader"
	"go.yaml.in/yaml/v2"
)

const unsupportedTroubleshootAnnotation = "kots.io/unsupported-troubleshoot"
const UnsupportedTroubleshootMessage = "Some v1beta3 Troubleshoot specs use unsupported Helm templates or preflight database references. Those specs will not run; other checks run normally."

var beta3TroubleshootVersion = regexp.MustCompile(`(?m)^apiVersion:[ \t]*['"]?troubleshoot\.sh/v1beta3['"]?[ \t]*(?:#.*)?$`)

func IsV1Beta3TroubleshootDoc(content []byte) bool {
	var header struct {
		APIVersion string `yaml:"apiVersion"`
	}
	if yaml.Unmarshal(content, &header) == nil {
		return header.APIVersion == "troubleshoot.sh/v1beta3"
	}
	// Unrendered Helm conditionals may not yet be valid YAML.
	return beta3TroubleshootVersion.MatchString(strings.ReplaceAll(string(content), "\r\n", "\n"))
}

// Persist the unsupported-version flag in the rendered archive, without checks.
func UnsupportedTroubleshootWarning() []byte {
	return []byte(`apiVersion: troubleshoot.sh/v1beta2
kind: Preflight
metadata:
  annotations:
    kots.io/unsupported-troubleshoot: "true"
`)
}

func LoadTSKindsFromPath(dir string) (*troubleshootloader.TroubleshootKinds, error) {
	if _, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrapf(err, "failed to stat dir: %s", dir)
		}
		return &troubleshootloader.TroubleshootKinds{}, nil
	}

	var renderedFiles []string
	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.Wrapf(err, "failed to walk dir: %s", dir)
			}

			if info.IsDir() {
				return nil
			}

			contents, err := os.ReadFile(path)
			if err != nil {
				return errors.Wrap(err, "failed to read file")
			}

			contents, err = PrepareSupportBundleReferences(contents)
			if err != nil {
				return errors.Wrap(err, "failed to prepare support bundle")
			}
			renderedFiles = append(renderedFiles, string(contents))

			return nil
		})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to walk dir: %s", dir)
	}

	ctx := context.Background()
	tsKinds, err := troubleshootloader.LoadSpecs(ctx, troubleshootloader.LoadOptions{
		RawSpecs: renderedFiles,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to load troubleshoot specs from archive")
	}
	return tsKinds, nil
}
