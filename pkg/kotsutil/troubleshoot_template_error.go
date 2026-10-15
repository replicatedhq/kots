package kotsutil

import (
	"regexp"
	"strings"
	"text/template/parse"

	"go.yaml.in/yaml/v2"
)

const unsupportedTroubleshootAnnotation = "kots.io/unsupported-troubleshoot"

const UnsupportedTroubleshootMessage = "Some v1beta3 Troubleshoot specs contain Helm templates that KOTS cannot render. Those specs will not run; other checks run normally."

var beta3TroubleshootVersion = regexp.MustCompile(`(?m)^apiVersion:[ \t]*['"]?troubleshoot\.sh/v1beta3['"]?[ \t]*(?:#.*)?$`)
var kotsTemplateExpression = regexp.MustCompile(`(?s)(?:repl\{\{|\{\{repl).*?\}\}`)

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

// Concrete beta3 specs use Troubleshoot's existing execution types. Standalone
// Helm templates still need values; preserve them and show a notice instead.
func IsUnsupportedTroubleshootDoc(content []byte) bool {
	if !IsV1Beta3TroubleshootDoc(content) {
		return false
	}
	content = kotsTemplateExpression.ReplaceAll(content, nil)
	t := parse.New("troubleshoot")
	t.Mode = parse.SkipFuncCheck | parse.ParseComments
	trees := map[string]*parse.Tree{}
	if _, err := t.Parse(string(content), "{{", "}}", trees); err != nil {
		// A document can reference a variable defined in another document.
		return strings.Contains(string(content), "{{") && (strings.Contains(string(content), "}}") || strings.Contains(string(content), ".Values"))
	}
	if len(trees) > 1 {
		return true
	}
	for _, node := range t.Root.Nodes {
		// Troubleshoot also expands runtime fields in analyzer messages, such
		// as {{ .version }}. These are not chart values to render here.
		if action, ok := node.(*parse.ActionNode); ok && len(action.Pipe.Cmds) == 1 && len(action.Pipe.Cmds[0].Args) == 1 {
			if field, ok := action.Pipe.Cmds[0].Args[0].(*parse.FieldNode); ok {
				switch field.Ident[0] {
				case "Values", "Chart", "Release", "Capabilities", "Files", "Template":
				default:
					continue
				}
			}
		}
		if _, text := node.(*parse.TextNode); !text {
			return true
		}
	}
	return false
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
