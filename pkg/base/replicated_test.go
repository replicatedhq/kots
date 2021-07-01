package base

import (
	"reflect"
	"testing"

	"github.com/replicatedhq/kots/pkg/template"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func Test_findAllKotsHelmCharts(t *testing.T) {
	tests := []struct {
		name    string
		content string
		expect  map[string]interface{}
	}{
		{
			name: "simple",
			content: `
apiVersion: "kots.io/v1beta1"
kind: "HelmChart"
metadata:
  name: "test"
spec:
  values:
    isStr: this is a string
    isKurlBool: repl{{ IsKurl }}
    isKurlStr: "repl{{ IsKurl }}"
    isBool: true
    nestedValues:
      isNumber1: 100
      isNumber2: 100.5
`,
			expect: map[string]interface{}{
				"isStr":      "this is a string",
				"isKurlBool": false,
				"isKurlStr":  "false",
				"isBool":     true,
				"nestedValues": map[string]interface{}{
					"isNumber1": float64(100),
					"isNumber2": float64(100.5),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			upstreamFiles := []upstreamtypes.UpstreamFile{
				{
					Path:    "/heml/chart.yaml",
					Content: []byte(test.content),
				},
			}

			builder := template.Builder{}
			builder.AddCtx(template.StaticCtx{})

			helmCharts, err := findAllKotsHelmCharts(upstreamFiles, builder, nil)
			req.NoError(err)
			assert.Len(t, helmCharts, 1)

			helmValues, err := helmCharts[0].Spec.GetHelmValues(helmCharts[0].Spec.Values)
			req.NoError(err)

			assert.Equal(t, test.expect, helmValues)
		})
	}
}

func Test_renderReplicated(t *testing.T) {
	// podName tests a normal ConfigOption, proving this process does not impact existing behavior
	// mountPath tests defaults if no value is provided
	// secretName-1 tests repeatable items with a string
	// secretName-2 tests repeatable items with an int
	// secretName-3 tests repeatable items with a bool
	// don't touch this! tests merging repeatable items with existing items
	tests := []struct {
		name          string
		upstream      *upstreamtypes.Upstream
		renderOptions *RenderOptions
		expectedFile  BaseFile
	}{
		{
			name: "replace array with repeat values",
			upstream: &upstreamtypes.Upstream{
				Files: []upstreamtypes.UpstreamFile{
					{
						Path: "config.yaml",
						Content: []byte(`
apiVersion: kots.io/v1beta1 
kind: Config 
metadata: 
  creationTimestamp: null 
  name: config-sample 
spec: 
  groups:
  - name: "podInfo"
    description: "info for pod"
    items:
    - name: "podName"
      type: "text"
      default: "test"
      value: "testPod"
    - name: "mountPath"
      type: "text"
      default: "/var/www/html"
  - name: secrets
    minimumCount: 1
    title: Secrets 
    description: Buncha Secrets
    items: 
    - name: "secretName"
      type: "text"
      title: "Secret Name"
      default: "onetwothree"
      repeatable: true
      minimumCount: 1
      templates:
      - apiVersion: apps/v1
        kind: Deployment
        name: my-deploy
        namespace: my-app
        yamlPath: spec.template.spec.containers[0].volumes[1].projected.sources[1]
`,
						),
					},
					{
						Path: "userdata/config.yaml",
						Content: []byte(`apiVersion: kots.io/v1beta1
kind: ConfigValues
metadata:
  name: test-app
spec:
  values:
    secretName-1:
      value: "123"
      repeatableItem: secretName
    secretName-2:
      value: "456"
      repeatableItem: secretName
    secretName-3:
      value: "789"
      repeatableItem: secretName
status: {}
`,
						),
					},
					{
						Path: "deployment.yaml",
						Content: []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deploy
  namespace: my-app
spec:
  template:
    spec:
      containers:
        - name: repl{{ ConfigOption "podName"}}
          image: httpd
          volumeMounts:
          - mountPath: '{{repl ConfigOption "mountPath"}}'
            name: secret-assets
            readOnly: true
          volumes:
          - name: normalVolume
            projected:
              sources:
              - name: normalSecret
          - name: secret-assets
            projected:
              sources:
              - secret:
                  name: "don't touch this!"
                  metaData:
                    fileName: 'repl{{ ConfigOptionName "secretName"}}'
              - secret:
                  name: 'repl{{ ConfigOption "secretName"}}'
                  pod: repl{{ ConfigOption "podName" }}
                  metaData:
                    pod: repl{{ ConfigOption "podName"}}
                    fileName: 'repl{{ ConfigOptionName "secretName"}}'
              - secret:
                  name: "don't touch this either!"`),
					},
				},
			},
			renderOptions: &RenderOptions{},
			expectedFile: BaseFile{
				Path: "deployment.yaml",
				Content: []byte(
					`apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deploy
  namespace: my-app
spec:
  template:
    spec:
      containers:
        - image: httpd
          name: testPod
          volumeMounts:
            - mountPath: /var/www/html
              name: secret-assets
              readOnly: true
          volumes:
            - name: normalVolume
              projected:
                sources:
                  - name: normalSecret
            - name: secret-assets
              projected:
                sources:
                  - secret:
                      name: "don't touch this!"
                      metaData:
                        fileName: "secretName"
                  - secret:
                      name: "don't touch this either!"
                  - secret:
                      name: "123"
                      pod: "testPod"
                      metaData:
                        pod: "testPod"
                        fileName: "secretName-1"
                  - secret:
                      name: "456"
                      pod: "testPod"
                      metaData:
                        pod: "testPod"
                        fileName: "secretName-2"
                  - secret:
                      name: "789"
                      pod: "testPod"
                      metaData:
                        pod: "testPod"
                        fileName: "secretName-3"`),
			},
		},
	}

	for _, test := range tests {
		base, err := renderReplicated(test.upstream, test.renderOptions)
		if err != nil {
			t.Errorf("renderReplicated(...) test %s failed with error %v", test.name, err)
		}

		for _, targetFile := range base.Files {
			if targetFile.Path == test.expectedFile.Path {
				var got interface{}
				var want interface{}
				err := yaml.Unmarshal(targetFile.Content, &got)
				if err != nil {
					t.Errorf("renderReplicated(...) test %s failed with error %v", test.name, err)
				}
				err = yaml.Unmarshal(test.expectedFile.Content, &want)
				if err != nil {
					t.Errorf("renderReplicated(...) test %s failed with error %v", test.name, err)
				}

				if !reflect.DeepEqual(got, want) {
					t.Errorf("renderReplicated(...) test %s failed: wanted \n---\n%s, got \n---\n%s", test.name, want, got)
				}
			}
		}
	}
}
