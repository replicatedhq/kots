package base

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/require"
)

func Test_buildStackFromYaml(t *testing.T) {
	tests := []struct {
		name      string
		yamlPath  string
		yaml      map[string]interface{}
		wantStack yamlStack
	}{
		{
			name:     "build stack",
			yamlPath: "spec.template.spec.containers[0]",
			yaml: map[string]interface{}{
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"spec": map[string]interface{}{
							"containers": []interface{}{
								map[string]interface{}{
									"name": "hello",
								},
							},
						},
					},
				},
			},
			wantStack: yamlStack{
				{
					NodeName: "",
					Type:     "map",
					Index:    0,
					Data: map[string]interface{}{
						"spec": map[string]interface{}{
							"template": map[string]interface{}{
								"spec": map[string]interface{}{
									"containers": []interface{}{
										map[string]interface{}{
											"name": "hello",
										},
									},
								},
							},
						},
					},
					Array: nil,
				},
				{
					NodeName: "spec",
					Type:     "map",
					Index:    0,
					Data: map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name": "hello",
									},
								},
							},
						},
					},
					Array: nil,
				},
				{
					NodeName: "template",
					Type:     "map",
					Index:    0,
					Data: map[string]interface{}{
						"spec": map[string]interface{}{
							"containers": []interface{}{
								map[string]interface{}{
									"name": "hello",
								},
							},
						},
					},
					Array: nil,
				},
				{
					NodeName: "spec",
					Type:     "map",
					Index:    0,
					Data: map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "hello",
							},
						},
					},
					Array: nil,
				},
				{
					NodeName: "containers",
					Type:     "array",
					Index:    0,
					Data: map[string]interface{}{
						"name": "hello",
					},
					Array: []interface{}{
						map[string]interface{}{
							"name": "hello",
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			result, err := buildStackFromYaml(test.yamlPath, test.yaml)
			req.NoError(err)

			req.Equal(test.wantStack, result)
		})
	}
}

func Test_buildYamlFromStack(t *testing.T) {
	tests := []struct {
		name     string
		stack    yamlStack
		wantyaml map[string]interface{}
	}{
		{
			name: "build stack",
			stack: yamlStack{
				{
					NodeName: "",
					Type:     "map",
					Index:    0,
					Data: map[string]interface{}{
						"spec": map[string]interface{}{
							"template": map[string]interface{}{
								"spec": map[string]interface{}{
									"containers": []interface{}{
										map[string]interface{}{
											"name": "hello",
										},
									},
								},
							},
						},
					},
					Array: nil,
				},
				{
					NodeName: "spec",
					Type:     "map",
					Index:    0,
					Data: map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name": "hello",
									},
								},
							},
						},
					},
					Array: nil,
				},
				{
					NodeName: "template",
					Type:     "map",
					Index:    0,
					Data: map[string]interface{}{
						"spec": map[string]interface{}{
							"containers": []interface{}{
								map[string]interface{}{
									"name": "hello",
								},
							},
						},
					},
					Array: nil,
				},
				{
					NodeName: "spec",
					Type:     "map",
					Index:    0,
					Data: map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "hello",
							},
						},
					},
					Array: nil,
				},
				{
					NodeName: "containers",
					Type:     "array",
					Index:    0,
					Data: map[string]interface{}{
						"name": "hello",
					},
					Array: []interface{}{
						map[string]interface{}{
							"name": "hello",
						},
					},
				},
			},
			wantyaml: map[string]interface{}{
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"spec": map[string]interface{}{
							"containers": []interface{}{
								map[string]interface{}{
									"name": "hello",
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

			result := buildYamlFromStack(test.stack)

			req.Equal(test.wantyaml, result)
		})
	}
}

func Test_parseVariadicTarget(t *testing.T) {
	tests := []struct {
		name             string
		configOptionName string
		valueName        string
		target           string
		want             interface{}
	}{
		{
			configOptionName: "secret",
			valueName:        "secret-123",
			target:           "repl{{ ConfigOption \"repl[[ .secret ]]\" }}",
			want:             "repl{{ ConfigOption \"secret-123\" }}",
		},
		{
			configOptionName: "ingress_hostname",
			valueName:        "ingress_hostname-123",
			target: `repl{{ $ca := genCA (ConfigOption "repl[[ .ingress_hostname ]]" ) 365 }}
			repl{{ $tls := dict "ca" $ca }}
			repl{{ $cert := genSignedCert (ConfigOption "repl[[ .ingress_hostname ]]" ) (list ) (list (ConfigOption "[[repl .ingress_hostname ]]" )) 365 $ca }}
			repl{{ $_ := set $tls "cert" $cert }}
			repl{{ toJson $tls }}`,
			want: `repl{{ $ca := genCA (ConfigOption "ingress_hostname-123" ) 365 }}
			repl{{ $tls := dict "ca" $ca }}
			repl{{ $cert := genSignedCert (ConfigOption "ingress_hostname-123" ) (list ) (list (ConfigOption "ingress_hostname-123" )) 365 $ca }}
			repl{{ $_ := set $tls "cert" $cert }}
			repl{{ toJson $tls }}`,
		},
		{
			configOptionName: "secret",
			valueName:        "secret-789",
			target:           "repl{{ ConfigOptionFilename '[[repl .secret ]]' }}",
			want:             "repl{{ ConfigOptionFilename 'secret-789' }}",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			result, err := parseVariadicTarget(test.configOptionName, test.valueName, test.target)
			req.NoError(err)

			req.Equal(test.want, result)
		})
	}
}

func Test_getUpstreamTemplateData(t *testing.T) {
	tests := []struct {
		name         string
		content      []byte
		wantMetadata kotsv1beta1.RepeatTemplate
		wantErr      bool
	}{
		{
			name: "has metadata",
			content: []byte(`
apiVersion: kots.io/v1beta1 
kind: Config 
metadata: 
  creationTimestamp: null 
  name: config-sample
  namespace: test`),
			wantMetadata: kotsv1beta1.RepeatTemplate{
				APIVersion: "kots.io/v1beta1",
				Kind:       "Config",
				Name:       "config-sample",
				Namespace:  "test",
			},
		},
		{
			name: "metadata in the wrong spot",
			content: []byte(`
apiVersion: kots.io/v1beta1 
kind: Config 
data: 
  creationTimestamp: null
  metadata:
    name: config-sample
    namespace: test`),
			wantMetadata: kotsv1beta1.RepeatTemplate{
				APIVersion: "kots.io/v1beta1",
				Kind:       "Config",
				Name:       "config-sample",
				Namespace:  "test",
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			metadataResult, _, err := getUpstreamTemplateData(test.content)
			if test.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
				req.Equal(test.wantMetadata, metadataResult)
			}
		})
	}
}
