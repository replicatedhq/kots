package config

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/stretchr/testify/require"
	"go.undefinedlabs.com/scopeagent"
)

func TestTemplateConfig(t *testing.T) {
	log := logger.NewLogger()
	log.Silence()

	tests := []struct {
		name             string
		configSpecData   string
		configValuesData string
		want             string
	}{
		{
			name: "basic, no template functions",
			configSpecData: `
apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: test-app
spec:
  groups:
    - name: example_settings
      title: My Example Config
      description: Configuration to serve as an example for creating your own
      items:
        - name: a_string
          title: a string field
          type: string
          default: "abc123"`,
			configValuesData: `
apiVersion: kots.io/v1beta1
kind: ConfigValues
metadata:
  name: test-app
spec:
  values:
    a_string:
      value: "xyz789"
status: {}
`,
			want: `kind: Config
apiVersion: kots.io/v1beta1
metadata:
  name: test-app
spec:
  groups:
  - name: example_settings
    title: My Example Config
    description: Configuration to serve as an example for creating your own
    items:
    - name: a_string
      type: string
      title: a string field
      default: ""
      value: xyz789
`,
		},
		{
			name: "one long 'when' template function",
			configSpecData: `
apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: test-app
spec:
  groups:
   - name: database_settings_group
     items:
     - name: db_type
       type: select_one
       default: embedded
       items:
       - name: external
         title: External
       - name: embedded
         title: Embedded DB
     - name: database_password
       title: Database Password
       type: password
       when: '{{repl or (ConfigOptionEquals "db_type" "external") (ConfigOptionEquals "db_type" "embedded")}}'`,
			configValuesData: `
apiVersion: kots.io/v1beta1
kind: ConfigValues
metadata:
  name: test-app
spec:
  values: {}
status: {}
`,
			want: `kind: Config
apiVersion: kots.io/v1beta1
metadata:
  name: test-app
spec:
  groups:
  - name: database_settings_group
    title: ""
    items:
    - name: db_type
      type: select_one
      default: embedded
      value: ""
      items:
      - name: external
        title: External
        value: false
      - name: embedded
        title: Embedded DB
        value: false
    - name: database_password
      type: password
      title: Database Password
      default: ""
      value: ""
      when: 'true'
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scopetest := scopeagent.StartTest(t)
			defer scopetest.End()

			req := require.New(t)

			got, err := TemplateConfig(log, tt.configSpecData, tt.configValuesData)
			req.NoError(err)

			req.Equal(tt.want, got)
		})
	}
}
