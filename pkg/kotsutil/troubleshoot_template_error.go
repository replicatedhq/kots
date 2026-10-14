package kotsutil

import (
	"regexp"
	"strings"

	"go.yaml.in/yaml/v2"
)

const unsupportedTroubleshootAnnotation = "kots.io/unsupported-troubleshoot"

const UnsupportedTroubleshootMessage = "This release contains v1beta3 Troubleshoot specs. KOTS does not support them, so they will not run."

var beta3TroubleshootVersion = regexp.MustCompile(`(?m)^apiVersion:[ \t]*['"]?troubleshoot\.sh/v1beta3['"]?[ \t]*(?:#.*)?$`)

func IsUnsupportedTroubleshootDoc(content []byte) bool {
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
