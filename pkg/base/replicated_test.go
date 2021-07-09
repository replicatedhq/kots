package base

import (
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
		expectedSecrets    []BaseFile
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
      value: "MTIz"
      repeatableItem: secretName
    secretName-2:
      value: "MTIz"
      repeatableItem: secretName
    secretName-3:
      value: "MTIz"
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
          - secret:
              name: 'repl{{ ConfigOptionName repl[[ .secretName ]] }}'
              pod: repl{{ ConfigOption "podName" }}
              metaData:
                pod: repl{{ ConfigOption "podName"}}
                fileName: 'repl{{ ConfigOptionName repl[[ .secretName ]] }}'
              items:
                - key: "data"
                  path: '{{repl ConfigOptionName repl[[ .secretName ]] }}'
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
  repl{{ ConfigOptionName repl[[ .secretName ]] }}: '{{repl ConfigOption repl[[ .secretName ]] }}'`),
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
                  name: "secretName-1"
                  pod: "testPod"
                  metaData:
                    pod: "testPod"
                    fileName: "secretName-1"
                  items:
                  - key: "file"
                    path: "secretName-1"
              - secret:
                  name: "secretName-2"
                  pod: "testPod"
                  metaData:
                    pod: "testPod"
                    fileName: "secretName-2"
                  items:
                  - key: "file"
                    path: "secretName-2"
              - secret:
                  name: "secretName-3"
                  pod: "testPod"
                  metaData:
                    pod: "testPod"
                    fileName: "secretName-3"
                  items:
                  - key: "file"
                    path: "secretName-3"`),
			},
			expectedSecrets: []BaseFile{
				{
					Path: "secret-rando.yaml",
					Content: []byte(
						`apiVersion: v1
kind: Secret
metadata:
  name: secretName-1
  namespace: "my-app"
type: Opaque
data:
  secretName-1: MTIz`),
				},
				{
					Path: "secret-rando.yaml",
					Content: []byte(
						`apiVersion: v1
kind: Secret
metadata:
  name: secretName-2
  namespace: "my-app"
type: Opaque
data:
  secretName-2: MTIz`),
				},
				{
					Path: "secret-rando.yaml",
					Content: []byte(
						`apiVersion: v1
kind: Secret
metadata:
  name: secretName-3
  namespace: "my-app"
type: Opaque
data:
  secretName-3: MTIz`),
				},
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

			var unmarshaledSecrets []*corev1.Secret
			for _, expectedSecret := range test.expectedSecrets {
				secobj, _, err := decode(expectedSecret.Content, nil, nil)
				req.NoError(err)

				unmarshaledSecrets = append(unmarshaledSecrets, secobj.(*corev1.Secret))
			}

			secretsFound := 0

			for _, targetFile := range base.Files {
				obj, gvk, err := decode(targetFile.Content, nil, nil)
				if err != nil {
					continue
				}

				if gvk.Kind == "deployment" {
					deployment := obj.(*appsv1.Deployment)

					assert.ElementsMatch(t, expectedDeployment.Spec.Template.Spec.Volumes[1].Projected.Sources, deployment.Spec.Template.Spec.Volumes[1].Projected.Sources)
				}

				if gvk.Kind == "Secret" {
					secretsFound++
					secret := obj.(*corev1.Secret)

					for _, unmarshaledSecret := range unmarshaledSecrets {
						if secret.GetObjectMeta().GetName() == unmarshaledSecret.GetObjectMeta().GetName() {
							assert.Equal(t, unmarshaledSecret, secret)
						}
					}
				}
			}
			req.Equal(secretsFound, len(unmarshaledSecrets))

		})
	}
}

func Test_removeFileFromUpstream(t *testing.T) {
	tests := []struct {
		name  string
		files []upstreamtypes.UpstreamFile
		path  string
		want  []upstreamtypes.UpstreamFile
	}{
		{
			name: "remove file",
			files: []upstreamtypes.UpstreamFile{
				{
					Path: "deployment.yaml",
				},
				{
					Path: "deployment-hhf928.yaml",
				},
				{
					Path: "secret.yaml",
				},
			},
			path: "deployment.yaml",
			want: []upstreamtypes.UpstreamFile{
				{
					Path: "secret.yaml",
				},
				{
					Path: "deployment-hhf928.yaml",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := removeFileFromUpstream(test.files, test.path)

			assert.Equal(t, test.want, result)
		})
	}
}
