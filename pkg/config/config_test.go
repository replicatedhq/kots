package config

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/template"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kotskinds/multitype"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	log := logger.NewCLILogger(ioutil.Discard)
	log.Silence()

	license := &kotsv1beta1.License{
		Spec: kotsv1beta1.LicenseSpec{
			LicenseID: "abcdef",
			AppSlug:   "my-app",
			Endpoint:  "http://localhost:30016",
			Entitlements: map[string]kotsv1beta1.EntitlementField{
				"expires_at": {
					Title:       "Expiration",
					Description: "License Expiration",
				},
				"has-product-2": {
					Title: "Has Product 2",
					Value: kotsv1beta1.EntitlementValue{
						Type:   kotsv1beta1.String,
						StrVal: "test",
					},
				},
				"is_vip": {
					Title: "Is VIP",
					Value: kotsv1beta1.EntitlementValue{
						Type:    kotsv1beta1.Bool,
						BoolVal: false,
					},
				},
				"num_seats": {
					Title: "Number Of Seats",
					Value: kotsv1beta1.EntitlementValue{
						Type:   kotsv1beta1.Int,
						IntVal: 10,
					},
				},
				"sdzf": {
					Title: "sdf",
					Value: kotsv1beta1.EntitlementValue{
						Type:   kotsv1beta1.Int,
						IntVal: 1,
					},
				},
				"test": {
					Title: "test",
					Value: kotsv1beta1.EntitlementValue{
						Type:   kotsv1beta1.String,
						StrVal: "123asd",
					},
				},
			},
		},
	}

	application := &kotsv1beta1.Application{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "Application",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "local",
		},
		Spec: kotsv1beta1.ApplicationSpec{
			Title: "My Application",
		},
	}

	versionInfo := &template.VersionInfo{
		Sequence:                 0,
		Cursor:                   "345",
		ChannelName:              "Stable",
		VersionLabel:             "1.2.3",
		IsRequired:               true,
		ReleaseNotes:             "",
		IsAirgap:                 false,
		ReplicatedRegistryDomain: "custom.registry.com",
		ReplicatedProxyDomain:    "custom.proxy.com",
	}
	appInfo := &template.ApplicationInfo{Slug: "my-app-1"}

	tests := []struct {
		name             string
		configSpecData   string
		configValuesData map[string]template.ItemValue
		useAppSpec       bool
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
			configValuesData: map[string]template.ItemValue{
				"a_string": {
					Value: "xyz789",
				},
			},
			useAppSpec: true,
			want: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
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
			configValuesData: map[string]template.ItemValue{},
			useAppSpec:       false,
			want: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
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
			configValuesData: map[string]template.ItemValue{
				"other": {
					Value: "xyz789",
				},
			},
			useAppSpec: false,
			want: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
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
		{
			name: "repeatable Items",
			configSpecData: `apiVersion: kots.io/v1beta1 
kind: Config 
metadata:  
  name: test-app
spec: 
  groups:
  - name: secrets
    title: Secrets 
    description: Buncha Secrets
    items: 
    - name: "secretName"
      type: "text"
      title: "Secret Name"
      default: "onetwothree"
      repeatable: true
      minimumCount: 1
      count: 0
      templates:
      - apiVersion: apps/v1
        kind: Deployment
        name: my-deploy
        namespace: my-app
        yamlPath: spec.template.spec.containers[0].volumes[1].projected.sources[1]
`,
			configValuesData: map[string]template.ItemValue{
				"secretName-1": {
					Value:          "123",
					RepeatableItem: "secretName",
				},
				"secretName-2": {
					Value:          "456",
					RepeatableItem: "secretName",
				},
				"secretName-3": {
					Value:          "789",
					RepeatableItem: "secretName",
				},
			},
			useAppSpec: true,
			want: `apiVersion: kots.io/v1beta1 
kind: Config 
metadata:
  name: test-app
spec: 
  groups:
  - name: secrets
    title: Secrets 
    description: Buncha Secrets
    items: 
    - name: "secretName"
      type: "text"
      title: "Secret Name"
      default: "onetwothree"
      repeatable: true
      minimumCount: 1
      countByGroup:
        secrets: 3
      templates:
      - apiVersion: apps/v1
        kind: Deployment
        name: my-deploy
        namespace: my-app
        yamlPath: spec.template.spec.containers[0].volumes[1].projected.sources[1]
      valuesByGroup: 
        secrets:
          secretName-1: "123"
          secretName-2: "456"
          secretName-3: "789"
`,
			expectOldFail: false,
		},
		{
			name: "non-BMP unicode in help_text with template function",
			configSpecData: `
apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: test-app
spec:
  groups:
    - name: example_settings
      title: My Example Config
      items:
        - name: other_field
          title: Other Field
          type: text
          default: "hello"
        - name: a_field
          title: A Field
          type: bool
          default: "1"
          help_text: "🔏 This field is locked repl{{ ConfigOption \"other_field\" }}"`,
			configValuesData: map[string]template.ItemValue{
				"other_field": {
					Value: "world",
				},
			},
			useAppSpec: false,
			want: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: test-app
spec:
  groups:
  - items:
    - default: "hello"
      name: other_field
      title: Other Field
      type: text
      value: world
    - default: "1"
      help_text: "🔏 This field is locked world"
      name: a_field
      title: A Field
      type: bool
      value: ""
    name: example_settings
    title: My Example Config
status: {}
`,
			expectOldFail: true,
		},
		{
			name: "non-BMP unicode in multi-line block-scalar help_text with repl template (SC-135815)",
			configSpecData: `
apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: test-app
spec:
  groups:
    - name: test_group
      title: Test
      items:
        - name: other_field
          title: Other Field
          type: bool
          default: "1"
        - name: test_field
          title: Test Field
          type: bool
          default: "1"
          help_text: |-
            Some description text

            🔏 **This field may not be edited once set**

            repl{{ if (ne Sequence 0) }}
            repl{{- if ConfigOptionEquals "other_field" "1" }}
            ℹ️ A conditional note
            repl{{- end }}
            repl{{- end }}`,
			configValuesData: map[string]template.ItemValue{
				"other_field": {
					Value: "1",
				},
			},
			useAppSpec: false,
			want: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: test-app
spec:
  groups:
  - items:
    - default: "1"
      name: other_field
      title: Other Field
      type: bool
      value: "1"
    - default: "1"
      help_text: |-
        Some description text

        🔏 **This field may not be edited once set**
      name: test_field
      title: Test Field
      type: bool
      value: ""
    name: test_group
    title: Test
status: {}
`,
			expectOldFail: true,
		},
		{
			name: "customer workaround: printf escape in repl template renders to non-BMP emoji",
			configSpecData: `
apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: sample-app
spec:
  groups:
    - name: sample-group
      title: Sample Title
      description: |
        sample description
      items:
        - name: dns_support_choice
          title: DNS Support
          type: text
          required: true
          help_text: |
             repl{{printf "\U0001F512"}}`,
			configValuesData: map[string]template.ItemValue{},
			useAppSpec:       false,
			want: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: sample-app
spec:
  groups:
  - description: |
      sample description
    items:
    - help_text: "🔒"
      name: dns_support_choice
      required: true
      title: DNS Support
      type: text
      value: ""
    name: sample-group
    title: Sample Title
status: {}
`,
			expectOldFail: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			kotsscheme.AddToScheme(scheme.Scheme)

			// in package myapigroupv1...
			decode := scheme.Codecs.UniversalDeserializer().Decode
			wantObj, _, err := decode([]byte(tt.want), nil, nil)
			req.NoError(err)

			var app *kotsv1beta1.Application
			if tt.useAppSpec {
				app = application
			}

			configObj, _, _ := decode([]byte(tt.configSpecData), nil, nil)

			localRegistry := registrytypes.RegistrySettings{}
			licenseWrapper := licensewrapper.LicenseWrapper{V1: license}
			got, err := templateConfigObjects(configObj.(*kotsv1beta1.Config), tt.configValuesData, &licenseWrapper, app, localRegistry, versionInfo, appInfo, nil, "app-namespace", false, MarshalConfig)
			req.NoError(err)

			gotObj, _, err := decode([]byte(got), nil, nil)
			req.NoError(err)

			req.Equal(wantObj, gotObj)

			// compare with oldMarshalConfig results
			got, err = templateConfigObjects(configObj.(*kotsv1beta1.Config), tt.configValuesData, &licenseWrapper, app, localRegistry, versionInfo, appInfo, nil, "app-namespace", false, oldMarshalConfig)
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

func TestApplyValuesToConfig(t *testing.T) {
	tests := []struct {
		name   string
		config kotsv1beta1.Config
		values map[string]template.ItemValue
		want   kotsv1beta1.Config
	}{
		{
			name: "create minimumCount",
			config: kotsv1beta1.Config{
				Spec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Name: "secrets",
							Items: []kotsv1beta1.ConfigItem{
								{
									Name:         "secretName",
									Repeatable:   true,
									MinimumCount: 1,
									CountByGroup: map[string]int{
										"secrets": 2,
									},
									ValuesByGroup: kotsv1beta1.ValuesByGroup{
										"secrets": {
											"secretName-1": "111",
											"secretName-2": "222",
										},
										// use this to test creating minimum count entries for a group
										// since it creates UUIDs, there's no way to test equality and the test will fail
										//"alsoSecrets": {},
									},
								},
							},
						},
						{
							Name: "pod",
							Items: []kotsv1beta1.ConfigItem{
								{
									Name:  "podName",
									Value: multitype.BoolOrString{Type: 0, StrVal: "real-pod"},
								},
							},
						},
					},
				},
			},
			values: map[string]template.ItemValue{
				"secretName-1": {
					Value:          "123",
					RepeatableItem: "secretName",
				},
				"secretName-2": {
					Value:          "456",
					RepeatableItem: "secretName",
				},
				"podName": {
					Value: "test-pod",
				},
			},
			want: kotsv1beta1.Config{
				Spec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Name: "secrets",
							Items: []kotsv1beta1.ConfigItem{
								{
									Name:         "secretName",
									Repeatable:   true,
									MinimumCount: 1,
									CountByGroup: map[string]int{
										"secrets": 2,
									},
									ValuesByGroup: kotsv1beta1.ValuesByGroup{
										"secrets": {
											"secretName-1": "123",
											"secretName-2": "456",
										},
									},
								},
							},
						},
						{
							Name: "pod",
							Items: []kotsv1beta1.ConfigItem{
								{
									Name:  "podName",
									Value: multitype.BoolOrString{Type: 0, StrVal: "test-pod"},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			resultConfig := ApplyValuesToConfig(&test.config, test.values)

			req.Equal(test.want, *resultConfig)
		})
	}
}

func TestDecodeUnicodeEscapes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "8-digit non-BMP escape",
			input: `\U0001F510`,
			want:  "\U0001F510",
		},
		{
			name:  "surrogate pair",
			input: `\uD83D\uDD10`,
			want:  "\U0001F510",
		},
		{
			name:  "4-digit BMP escape",
			input: `\u00E9`,
			want:  "\u00E9",
		},
		{
			name:  "no escapes unchanged",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "invalid codepoint left unchanged",
			input: `\UFFFFFFFF`,
			want:  `\UFFFFFFFF`,
		},
		{
			name:  "lone surrogate left unchanged",
			input: `\uD800`,
			want:  `\uD800`,
		},
		{
			name:  "mixed content preserved",
			input: "text before \\U0001F510 text after",
			want:  "text before \U0001F510 text after",
		},
		{
			name:  "multiple escapes decoded",
			input: `\U0001F510 and \U0001F512`,
			want:  "\U0001F510 and \U0001F512",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeUnicodeEscapes(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}
