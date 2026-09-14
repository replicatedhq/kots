package validators

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/lint/types"
	"github.com/stretchr/testify/require"
)

func lintExpressionsForRule(lintExpressions []types.LintExpression, rule string) []types.LintExpression {
	matched := []types.LintExpression{}
	for _, lintExpression := range lintExpressions {
		if lintExpression.Rule == rule {
			matched = append(matched, lintExpression)
		}
	}
	return matched
}

func TestValidateOPANonRendered_ConfigOptionNames(t *testing.T) {
	require.NoError(t, InitOPA())

	configFile := types.SpecFile{
		Name: "config.yaml",
		Path: "config.yaml",
		Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: app-config
spec:
  groups:
    - name: settings
      title: Application Settings
      items:
        - name: foo.bar.baz
          title: Dotted Name Item
          type: bool
          default: "1"
        - name: hostname
          title: Hostname
          type: text
          default: example.com
`,
	}

	tests := []struct {
		name             string
		specFiles        types.SpecFiles
		rule             string
		expectedMessages []string
	}{
		{
			name: "dotted config option name that exists is not reported as not found",
			specFiles: types.SpecFiles{
				configFile,
				{
					Name: "kots-app.yaml",
					Path: "kots-app.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app
spec:
  title: App
  statusInformers:
    - '{{repl if ConfigOptionEquals "foo.bar.baz" "1"}}deployment/x{{repl end}}'
`,
				},
			},
			rule:             "config-option-not-found",
			expectedMessages: nil,
		},
		{
			name: "missing dotted config option name is reported with its full name",
			specFiles: types.SpecFiles{
				configFile,
				{
					Name: "kots-app.yaml",
					Path: "kots-app.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app
spec:
  title: App
  statusInformers:
    - '{{repl if ConfigOptionEquals "does.not.exist" "1"}}deployment/x{{repl end}}'
`,
				},
			},
			rule:             "config-option-not-found",
			expectedMessages: []string{`Config option "does.not.exist" not found`},
		},
		{
			name: "dotted config option referencing itself is reported as circular",
			specFiles: types.SpecFiles{
				{
					Name: "config.yaml",
					Path: "config.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: app-config
spec:
  groups:
    - name: settings
      title: Application Settings
      items:
        - name: foo.bar.baz
          title: Circular Item
          type: text
          default: 'repl{{ ConfigOption "foo.bar.baz" }}'
`,
				},
			},
			rule:             "config-option-is-circular",
			expectedMessages: []string{`Config option "foo.bar.baz" references itself`},
		},
		{
			name: "sub-templated dotted config option is checked for repeatability with its full name",
			specFiles: types.SpecFiles{
				{
					Name: "config.yaml",
					Path: "config.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: app-config
spec:
  groups:
    - name: settings
      title: Application Settings
      items:
        - name: foo.bar.baz
          title: Dotted Name Item
          type: bool
          default: "1"
        - name: repeat.value
          title: Dotted Repeatable Item
          type: text
          repeatable: true
          templates:
          - name: example-config
          valuesByGroup:
            settings:
              key: value
`,
				},
				{
					Name: "configmap.yaml",
					Path: "configmap.yaml",
					Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config
data:
  ENV_VAR_1: '{{repl ConfigOption "[[repl .repeat.value ]]" }}'
  ENV_VAR_2: '{{repl ConfigOption "[[repl .foo.bar.baz ]]" }}'
`,
				},
			},
			rule:             "config-option-not-repeatable",
			expectedMessages: []string{`Config option "foo.bar.baz" not repeatable`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lintExpressions, err := ValidateOPANonRendered(tt.specFiles)
			require.NoError(t, err)

			actualMessages := []string{}
			for _, lintExpression := range lintExpressionsForRule(lintExpressions, tt.rule) {
				actualMessages = append(actualMessages, lintExpression.Message)
			}

			if len(tt.expectedMessages) == 0 {
				require.Empty(t, actualMessages)
			} else {
				require.Equal(t, tt.expectedMessages, actualMessages)
			}
		})
	}
}
