package config

import (
	"bytes"
	"io"
	"testing"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/template"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kotskinds/multitype"
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
	log := logger.NewCLILogger(io.Discard)
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
			configValuesData: map[string]template.ItemValue{},
			useAppSpec:       false,
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
			configValuesData: map[string]template.ItemValue{
				"other": {
					Value: "xyz789",
				},
			},
			useAppSpec: false,
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
		{
			name: "repeatable Items",
			configSpecData: `apiVersion: kots.io/v1beta1 
kind: Config 
metadata: 
  creationTimestamp: null 
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
  creationTimestamp: null 
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
			got, err := templateConfigObjects(configObj.(*kotsv1beta1.Config), tt.configValuesData, license, app, localRegistry, versionInfo, appInfo, nil, "app-namespace", false, MarshalConfig)
			req.NoError(err)

			gotObj, _, err := decode([]byte(got), nil, nil)
			req.NoError(err)

			req.Equal(wantObj, gotObj)

			// compare with oldMarshalConfig results
			got, err = templateConfigObjects(configObj.(*kotsv1beta1.Config), tt.configValuesData, license, app, localRegistry, versionInfo, appInfo, nil, "app-namespace", false, oldMarshalConfig)
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
