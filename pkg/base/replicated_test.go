package base

import (
	"fmt"
	"testing"

	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/client-go/kubernetes/scheme"
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
	// randomfile.yaml proves a file that cannot be marshaled will not throw an impactful error
	// bogusmetadata.yaml proves missing or malformed metadata will not throw an impactful error
	// podName tests a normal ConfigOption, proving this process does not impact existing behavior
	// mountPath tests defaults if no value is provided
	// secretName-1 tests repeatable items with a string
	// secretName-2 tests repeatable items with an int
	// secretName-3 tests repeatable items with a bool
	// don't touch this! tests merging repeatable items with existing items
	tests := []struct {
		name               string
		upstream           *upstreamtypes.Upstream
		renderOptions      *RenderOptions
		expectedDeployment BaseFile
		expectedSecret     BaseFile
	}{
		{
			name: "replace array with repeat values",
			upstream: &upstreamtypes.Upstream{
				Files: []upstreamtypes.UpstreamFile{
					{
						Path: "randomfile.yaml",
						Content: []byte(`
apiVersion:
  thisshouldntbehere:
    heyoooo
kind: 6
`),
					},
					{
						Path: "bogusmetadata.yaml",
						Content: []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name:
    badnesting: true
  namespace: 5
`),
					},
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
        yamlPath: spec.template.spec.volumes[1].projected.sources[1]
      - apiVersion: v1
        kind: Secret
        name: my-secret
        namespace: my-app
        yamlPath:
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
              items:
                - key: "data"
                  path: 'my-secrets/{{repl ConfigOption "secretName"}}'
          - secret:
              name: "don't touch this either!"`),
					},
					{
						Path: "secret.yaml",
						Content: []byte(
							`apiVersion: v1
kind: Secret
metadata:
  name: my-secret
  namespace: my-app
type: Opaque
data:
  password: MTIz`),
					},
				},
			},
			renderOptions: &RenderOptions{
				Log: logger.NewCLILogger(),
			},
			expectedDeployment: BaseFile{
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
                  items:
                  - key: "data"
                    path: "my-secrets/123"
              - secret:
                  name: "456"
                  pod: "testPod"
                  metaData:
                    pod: "testPod"
                    fileName: "secretName-2"
                  items:
                  - key: "data"
                    path: "my-secrets/456"
              - secret:
                  name: "789"
                  pod: "testPod"
                  metaData:
                    pod: "testPod"
                    fileName: "secretName-3"
                  items:
                  - key: "data"
                    path: "my-secrets/789"`),
			},
			expectedSecret: BaseFile{
				Path: "secret.yaml",
				Content: []byte(
					`apiVersion: v1
kind: Secret
metadata:
  name: my-secret
type: Opaque
data:
  data: MTIz`),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			base, err := renderReplicated(test.upstream, test.renderOptions)
			req.NoError(err)

			decode := scheme.Codecs.UniversalDeserializer().Decode
			depobj, _, err := decode(test.expectedDeployment.Content, nil, nil)
			req.NoError(err)

			expectedDeployment := depobj.(*appsv1.Deployment)

			secobj, _, err := decode(test.expectedSecret.Content, nil, nil)
			req.NoError(err)

			expectedSecret := secobj.(*corev1.Secret)

			for _, targetFile := range base.Files {
				obj, gvk, err := decode(targetFile.Content, nil, nil)
				if err != nil {
					continue
				}
				fmt.Printf("obj: %+v\n\n", obj)

				if gvk.Kind == "deployment" {
					deployment := obj.(*appsv1.Deployment)

					assert.ElementsMatch(t, expectedDeployment.Spec.Template.Spec.Volumes[1].Projected.Sources, deployment.Spec.Template.Spec.Volumes[1].Projected.Sources)
				}

				if gvk.Kind == "secret" {
					secret := obj.(*corev1.Secret)

					fmt.Printf("secret is %+v\n", secret)

					assert.Equal(t, expectedSecret, secret)
				}
			}
			assert.Fail(t, "")

		})
	}
}
