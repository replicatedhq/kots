package config

import (
	"bytes"
	"testing"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

// the old config marshal function, preserved to allow validation
func oldMarshalConfig(config *kotsv1beta1.Config) (string, error) {
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var marshalled bytes.Buffer
	if err := s.Encode(config, &marshalled); err != nil {
		return "", errors.Wrap(err, "failed to marshal config")
	}
	return string(marshalled.Bytes()), nil
}

func TestTemplateConfig(t *testing.T) {
	log := logger.NewCLILogger()
	log.Silence()

	licenseData := `
apiVersion: kots.io/v1beta1
kind: License
metadata:
  name: local
spec:
  licenseID: abcdef
  appSlug: my-app
  endpoint: 'http://localhost:30016'
  entitlements:
    expires_at:
      title: Expiration
      description: License Expiration
      value: ""
    has-product-2:
      title: Has Product 2
      value: "test"
    is_vip:
      title: Is VIP
      value: false
    num_seats:
      title: Number Of Seats
      value: 10
    sdzf:
      title: sdf
      value: 1
    test:
      title: test
      value: "123asd"
  signature: IA==`

	tests := []struct {
		name             string
		configSpecData   string
		configValuesData string
		want             string
		expectOldFail    bool
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
          type: text
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
			want: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  creationTimestamp: null
  name: test-app
spec:
  groups:
  - description: Configuration to serve as an example for creating your own
    items:
    - default: "abc123"
      name: a_string
      title: a string field
      type: text
      value: xyz789
    name: example_settings
    title: My Example Config
status: {}
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
			want: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  creationTimestamp: null
  name: test-app
spec:
  groups:
  - items:
    - default: embedded
      items:
      - value: ""
        default: ""
        name: external
        title: External
      - value: ""
        default: ""
        name: embedded
        title: Embedded DB
      name: db_type
      type: select_one
      value: ""
    - default: ""
      name: database_password
      title: Database Password
      type: password
      value: ""
      when: 'true'
    name: database_settings_group
    title: ""
status: {}
`,
			expectOldFail: false,
		},
		{
			name: "one long 'value' template function",
			configSpecData: `
apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: test-app
spec:
  groups:
   - name: test_value
     items:
     - name: test_title
       type: label
       title: repl{{ ConfigOption "other" }}
     - name: test_text
       type: text
       title: repl{{ ConfigOption "other" }}
       value: repl{{ ConfigOption "other" }}
     - name: other
       title: other
       type: text
       default: 'val1'`,
			configValuesData: `
apiVersion: kots.io/v1beta1
kind: ConfigValues
metadata:
  name: test-app
spec:
  values:
    other:
      value: "xyz789"
status: {}
`,
			want: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  creationTimestamp: null
  name: test-app
spec:
  groups:
  - items:
    - default: ""
      name: test_title
      title: xyz789
      type: label
      value: ""
    - default: ""
      name: test_text
      title: xyz789
      type: text
      value: "xyz789"
    - default: "val1"
      name: other
      title: other
      type: text
      value: xyz789
    name: test_value
    title: ""
status: {}
`,
			expectOldFail: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			decode := scheme.Codecs.UniversalDeserializer().Decode
			wantObj, _, err := decode([]byte(tt.want), nil, nil)
			req.NoError(err)

			localRegistry := template.LocalRegistry{}
			got, err := templateConfig(log, tt.configSpecData, tt.configValuesData, licenseData, "", localRegistry, "", MarshalConfig)
			req.NoError(err)

			gotObj, _, err := decode([]byte(got), nil, nil)
			req.NoError(err)

			req.Equal(wantObj, gotObj)

			// compare with oldMarshalConfig results
			got, err = templateConfig(log, tt.configSpecData, tt.configValuesData, licenseData, "", localRegistry, "", oldMarshalConfig)
			if !tt.expectOldFail {
				req.NoError(err)

				gotObj, _, err = decode([]byte(got), nil, nil)
				req.NoError(err)

				req.Equal(wantObj, gotObj)
			} else {
				req.Error(err)
			}
		})
	}
}
