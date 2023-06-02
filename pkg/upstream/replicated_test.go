package upstream

import (
	"reflect"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/multitype"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_releaseToFiles(t *testing.T) {
	tests := []struct {
		name     string
		release  *Release
		expected []types.UpstreamFile
	}{
		{
			name: "with common prefix",
			release: &Release{
				Manifests: map[string][]byte{
					"manifests/deployment.yaml": []byte("a: b"),
					"manifests/service.yaml":    []byte("c: d"),
				},
			},
			expected: []types.UpstreamFile{
				types.UpstreamFile{
					Path:    "deployment.yaml",
					Content: []byte("a: b"),
				},
				types.UpstreamFile{
					Path:    "service.yaml",
					Content: []byte("c: d"),
				},
			},
		},
		{
			name: "without common prefix",
			release: &Release{
				Manifests: map[string][]byte{
					"manifests/deployment.yaml": []byte("a: b"),
					"service.yaml":              []byte("c: d"),
				},
			},
			expected: []types.UpstreamFile{
				types.UpstreamFile{
					Path:    "manifests/deployment.yaml",
					Content: []byte("a: b"),
				},
				types.UpstreamFile{
					Path:    "service.yaml",
					Content: []byte("c: d"),
				},
			},
		},
		{
			name: "common prefix, with userdata",
			release: &Release{
				Manifests: map[string][]byte{
					"manifests/deployment.yaml": []byte("a: b"),
					"manifests/service.yaml":    []byte("c: d"),
					"userdata/values.yaml":      []byte("d: e"),
				},
			},
			expected: []types.UpstreamFile{
				types.UpstreamFile{
					Path:    "deployment.yaml",
					Content: []byte("a: b"),
				},
				types.UpstreamFile{
					Path:    "service.yaml",
					Content: []byte("c: d"),
				},
				types.UpstreamFile{
					Path:    "userdata/values.yaml",
					Content: []byte("d: e"),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			actual, err := releaseToFiles(test.release)
			req.NoError(err)

			assert.ElementsMatch(t, test.expected, actual)
		})
	}
}

func Test_createConfigValues(t *testing.T) {
	applicationName := "Test App"
	appInfo := &template.ApplicationInfo{Slug: "app-slug"}

	config := &kotsv1beta1.Config{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "Config",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: applicationName,
		},
		Spec: kotsv1beta1.ConfigSpec{
			Groups: []kotsv1beta1.ConfigGroup{
				kotsv1beta1.ConfigGroup{
					Name:  "group_name",
					Title: "Group Title",
					Items: []kotsv1beta1.ConfigItem{
						// should replace default
						kotsv1beta1.ConfigItem{
							Name: "1_with_default",
							Type: "string",
							Default: multitype.BoolOrString{
								Type:   multitype.String,
								StrVal: "default_1_new",
							},
							Value: multitype.BoolOrString{
								Type:   multitype.String,
								StrVal: "",
							},
						},
						// should preserve value and add default
						kotsv1beta1.ConfigItem{
							Name: "2_with_value",
							Type: "string",
							Default: multitype.BoolOrString{
								Type:   multitype.String,
								StrVal: "default_2",
							},
							Value: multitype.BoolOrString{
								Type:   multitype.String,
								StrVal: "value_2_new",
							},
						},
						// should add a new item
						kotsv1beta1.ConfigItem{
							Name: "4_with_default",
							Type: "string",
							Default: multitype.BoolOrString{
								Type:   multitype.String,
								StrVal: "default_4",
							},
						},
					},
				},
			},
		},
	}

	configValues := &kotsv1beta1.ConfigValues{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "ConfigValues",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: applicationName,
		},
		Spec: kotsv1beta1.ConfigValuesSpec{
			Values: map[string]kotsv1beta1.ConfigValue{
				"1_with_default": kotsv1beta1.ConfigValue{
					Default: "default_1",
				},
				"2_with_value": kotsv1beta1.ConfigValue{
					Value: "value_2",
				},
				"3_with_both": kotsv1beta1.ConfigValue{
					Value:   "value_3",
					Default: "default_3",
				},
			},
		},
	}

	req := require.New(t)

	// like new install, should match config
	expected1 := map[string]kotsv1beta1.ConfigValue{
		"1_with_default": kotsv1beta1.ConfigValue{
			Default: "default_1_new",
		},
		"2_with_value": kotsv1beta1.ConfigValue{
			Value:   "value_2_new",
			Default: "default_2",
		},
		"4_with_default": kotsv1beta1.ConfigValue{
			Default: "default_4",
		},
	}
	values1, err := createConfigValues(applicationName, config, nil, nil, nil, appInfo, nil, registrytypes.RegistrySettings{}, nil)
	req.NoError(err)
	assert.Equal(t, expected1, values1.Spec.Values)

	// Like an app without a config, should have exact same values
	expected2 := configValues.Spec.Values
	values2, err := createConfigValues(applicationName, nil, configValues, nil, nil, appInfo, nil, registrytypes.RegistrySettings{}, nil)
	req.NoError(err)
	assert.Equal(t, expected2, values2.Spec.Values)

	// updating existing values with new config, should do a merge
	expected3 := map[string]kotsv1beta1.ConfigValue{
		"1_with_default": kotsv1beta1.ConfigValue{
			Default: "default_1_new",
		},
		"2_with_value": kotsv1beta1.ConfigValue{
			Value:   "value_2",
			Default: "default_2",
		},
		"3_with_both": kotsv1beta1.ConfigValue{
			Value:   "value_3",
			Default: "default_3",
		},
		"4_with_default": kotsv1beta1.ConfigValue{
			Default: "default_4",
		},
	}
	values3, err := createConfigValues(applicationName, config, configValues, nil, nil, appInfo, nil, registrytypes.RegistrySettings{}, nil)
	req.NoError(err)
	assert.Equal(t, expected3, values3.Spec.Values)
}

func Test_findConfigInRelease(t *testing.T) {
	type args struct {
		release *Release
	}
	tests := []struct {
		name string
		args args
		want *kotsv1beta1.Config
	}{
		{
			name: "find config in single file release",
			args: args{
				release: &Release{
					Manifests: map[string][]byte{
						"filepath": []byte(`apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: config-sample
spec:
  groups:
  - name: example_settings
    title: My Example Config
    items:
    - name: show_text_inputs
      title: Customize Text Inputs
      help_text: "Show custom user text inputs"
      type: bool
`),
					},
				},
			},
			want: &kotsv1beta1.Config{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kots.io/v1beta1",
					Kind:       "Config",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "config-sample",
				},
				Spec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Name:        "example_settings",
							Title:       "My Example Config",
							Description: "",
							Items: []kotsv1beta1.ConfigItem{
								{
									Name:     "show_text_inputs",
									Type:     "bool",
									Title:    "Customize Text Inputs",
									HelpText: "Show custom user text inputs",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "find config in multidoc release",
			args: args{
				release: &Release{
					Manifests: map[string][]byte{
						"filepath": []byte(`apiVersion: app.k8s.io/v1beta1
kind: Application
metadata:
	name: "sample-app"
spec:
	descriptor:
	links:
		- description: Open App
		# needs to match applicationUrl in kots-app.yaml
		url: "http://sample-app"
---
apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: config-sample
spec:
  groups:
  - name: example_settings
    title: My Example Config
    items:
    - name: show_text_inputs
      title: Customize Text Inputs
      help_text: "Show custom user text inputs"
      type: bool
---
apiVersion: troubleshoot.sh/v1beta2
kind: SupportBundle
metadata:
name: support-bundle
spec:
collectors:
	- clusterInfo: {}
	- clusterResources: {}
	- logs:
		selector:
		- app=sample-app
		namespace: '{{repl Namespace }}'
`),
					},
				},
			},
			want: &kotsv1beta1.Config{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kots.io/v1beta1",
					Kind:       "Config",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "config-sample",
				},
				Spec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Name:        "example_settings",
							Title:       "My Example Config",
							Description: "",
							Items: []kotsv1beta1.ConfigItem{
								{
									Name:     "show_text_inputs",
									Type:     "bool",
									Title:    "Customize Text Inputs",
									HelpText: "Show custom user text inputs",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "find config in release with empty manifest",
			args: args{
				release: &Release{
					Manifests: map[string][]byte{
						"filepath": []byte(``),
					},
				},
			},
			want: nil,
		},
		{
			name: "find config with invalid yaml",
			args: args{
				release: &Release{
					Manifests: map[string][]byte{
						"filepath": []byte(`apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: config-sample
spec:
  groups:
  - name: example_settings
    title: My Example Config
    items:
    - name: show_text_inputs
      title: Customize Text Inputs
      help_text: "Show custom user text inputs"
      type: bool
   invalid_key: invalid_value
`),
					},
				},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findConfigInRelease(tt.args.release); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findConfigInRelease() = %v, want %v", got, tt.want)
			}

		})
	}
}
