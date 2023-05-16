package base

import (
	_ "embed"
	"io/ioutil"
	"reflect"
	"testing"

	envsubst "github.com/drone/envsubst/v2"
	"github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func Test_findAllKotsV1Beta1HelmCharts(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  []v1beta1.HelmChart
	}{
		{
			name: "one v1beta1 chart",
			files: map[string]string{
				"my-v1beta1-chart.yaml": `
apiVersion: "kots.io/v1beta1"
kind: "HelmChart"
metadata:
  name: "test"
spec:
  chart:
    name: "test"
  values: {}
`,
			},
			want: []v1beta1.HelmChart{
				{
					TypeMeta:   v1.TypeMeta{Kind: "HelmChart", APIVersion: "kots.io/v1beta1"},
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name: "test",
						},
						Values: map[string]v1beta1.MappedChartValue{},
					},
				},
			},
		},
		{
			name: "two v1beta1 charts",
			files: map[string]string{
				"my-v1beta1-chart.yaml": `
apiVersion: "kots.io/v1beta1"
kind: "HelmChart"
metadata:
  name: "test"
spec:
  chart:
    name: "test"
  values: {}
`,
				"my-other-v1beta1-chart.yaml": `
apiVersion: "kots.io/v1beta1"
kind: "HelmChart"
metadata:
  name: "test-2"
spec:
  chart:
    name: "test-2"
  values: {}
`,
			},
			want: []v1beta1.HelmChart{
				{
					TypeMeta:   v1.TypeMeta{Kind: "HelmChart", APIVersion: "kots.io/v1beta1"},
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name: "test",
						},
						Values: map[string]v1beta1.MappedChartValue{},
					},
				},
				{
					TypeMeta:   v1.TypeMeta{Kind: "HelmChart", APIVersion: "kots.io/v1beta1"},
					ObjectMeta: metav1.ObjectMeta{Name: "test-2"},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name: "test-2",
						},
						Values: map[string]v1beta1.MappedChartValue{},
					},
				},
			},
		},
		{
			name: "one v1beta1 chart with a matching v1beta2 chart",
			files: map[string]string{
				"my-v1beta1-chart.yaml": `
apiVersion: "kots.io/v1beta1"
kind: "HelmChart"
metadata:
  name: "test"
spec:
  chart:
    name: "test"
  values: {}
`,
				"my-v1beta2-chart.yaml": `
apiVersion: "kots.io/v1beta2"
kind: "HelmChart"
metadata:
  name: "test"
spec:
  chart:
    name: "test"
  values: {}
`,
			},
			want: []v1beta1.HelmChart{},
		},
		{
			name: "two v1beta1 charts with one matching v1beta2 chart",
			files: map[string]string{
				"my-v1beta1-chart.yaml": `
apiVersion: "kots.io/v1beta1"
kind: "HelmChart"
metadata:
  name: "test"
spec:
  chart:
    name: "test"
  values: {}
`,
				"my-other-v1beta1-chart.yaml": `
apiVersion: "kots.io/v1beta1"
kind: "HelmChart"
metadata:
  name: "test-2"
spec:
  chart:
    name: "test-2"
  values: {}
`,
				"my-v1beta2-chart.yaml": `
apiVersion: "kots.io/v1beta2"
kind: "HelmChart"
metadata:
  name: "test"
spec:
  chart:
    name: "test"
  values: {}
`,
			},
			want: []v1beta1.HelmChart{
				{
					TypeMeta:   v1.TypeMeta{Kind: "HelmChart", APIVersion: "kots.io/v1beta1"},
					ObjectMeta: metav1.ObjectMeta{Name: "test-2"},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name: "test-2",
						},
						Values: map[string]v1beta1.MappedChartValue{},
					},
				},
			},
		},
		{
			name: "one v1beta2 chart",
			files: map[string]string{
				"my-v1beta2-chart.yaml": `
apiVersion: "kots.io/v1beta2"
kind: "HelmChart"
metadata:
  name: "test"
spec:
  chart:
    name: "test"
  values: {}
`,
			},
			want: []v1beta1.HelmChart{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			upstreamFiles := []upstreamtypes.UpstreamFile{}
			for path, content := range test.files {
				upstreamFiles = append(upstreamFiles, upstreamtypes.UpstreamFile{
					Path:    path,
					Content: []byte(content),
				})
			}

			got, err := findAllKotsV1Beta1HelmCharts(upstreamFiles, template.Builder{}, nil)
			req.NoError(err)
			assert.ElementsMatch(t, test.want, got)
		})
	}
}

func Test_GetHelmValues(t *testing.T) {
	tests := []struct {
		name    string
		content string
		path    string
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
			path: "/helm/chart.yaml",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			upstreamFiles := []upstreamtypes.UpstreamFile{
				{
					Path:    test.path,
					Content: []byte(test.content),
				},
			}

			builder := template.Builder{}
			builder.AddCtx(template.StaticCtx{})

			helmCharts, err := findAllKotsV1Beta1HelmCharts(upstreamFiles, builder, nil)
			req.NoError(err)
			assert.Len(t, helmCharts, 1)

			helmValues, err := helmCharts[0].Spec.GetHelmValues(helmCharts[0].Spec.Values)
			req.NoError(err)

			assert.Equal(t, test.expect, helmValues)
		})
	}
}

func Test_renderReplicatedHelmBase(t *testing.T) {
	type args struct {
		u             *upstreamtypes.Upstream
		renderOptions *RenderOptions
		builder       template.Builder
		helmBase      Base
	}
	tests := []struct {
		name    string
		args    args
		want    *Base
		wantErr bool
	}{
		{
			name: "basic",
			args: args{
				u:             &upstreamtypes.Upstream{},
				renderOptions: &RenderOptions{SplitMultiDocYAML: true},
				builder:       template.Builder{},
				helmBase: Base{
					Path: "charts/my-chart",
					Files: []BaseFile{
						{
							Path:    "templates/deploy-1.yaml",
							Content: []byte("apiVersion: kots.io/v1beta1\nkind: File\nName: 1"),
						},
					},
					Bases: []Base{
						{
							Path: "crds",
							Files: []BaseFile{
								{
									Path:    "crd-1.yaml",
									Content: []byte("apiVersion: apiextensions.k8s.io/v1beta1\nkind: CustomResourceDefinition\nName: 2"),
								},
							},
						},
						{
							Path: "charts/my-subchart-1",
							Files: []BaseFile{
								{
									Path:    "templates/deploy-2.yaml",
									Content: []byte("apiVersion: kots.io/v1beta1\nkind: File\nName: 3"),
								},
							},
						},
						{
							Path: "charts/my-subchart-2",
							Files: []BaseFile{
								{
									Path:    "templates/deploy-3.yaml",
									Content: []byte("apiVersion: kots.io/v1beta1\nkind: File\nName: 4"),
								},
								{
									Path:    "templates/deploy-4.yaml",
									Content: []byte("apiVersion: kots.io/v1beta1\nkind: File\nName: 5"),
								},
							},
							Bases: []Base{
								{
									Path: "crds",
									Files: []BaseFile{
										{
											Path:    "crd-2.yaml",
											Content: []byte("apiVersion: apiextensions.k8s.io/v1beta1\nkind: CustomResourceDefinition\nName: 6"),
										},
									},
								},
								{
									Path: "charts/my-sub-subchart-1",
									Files: []BaseFile{
										{
											Path:    "templates/deploy-5.yaml",
											Content: []byte("apiVersion: kots.io/v1beta1\nkind: File\nName: 7"),
										},
									},
								},
							},
						},
					},
				},
			},
			want: &Base{
				Path: "charts/my-chart",
				Files: []BaseFile{
					{
						Path:    "templates/deploy-1.yaml",
						Content: []byte("apiVersion: kots.io/v1beta1\nkind: File\nName: 1"),
					},
				},
				ErrorFiles: []BaseFile{},
				Bases: []Base{
					{
						Path: "crds",
						Files: []BaseFile{
							{
								Path:    "crd-1.yaml",
								Content: []byte("apiVersion: apiextensions.k8s.io/v1beta1\nkind: CustomResourceDefinition\nName: 2"),
							},
						},
						ErrorFiles: []BaseFile{},
						Bases:      []Base{},
					},
					{
						Path: "charts/my-subchart-1",
						Files: []BaseFile{
							{
								Path:    "templates/deploy-2.yaml",
								Content: []byte("apiVersion: kots.io/v1beta1\nkind: File\nName: 3"),
							},
						},
						ErrorFiles: []BaseFile{},
						Bases:      []Base{},
					},
					{
						Path: "charts/my-subchart-2",
						Files: []BaseFile{
							{
								Path:    "templates/deploy-3.yaml",
								Content: []byte("apiVersion: kots.io/v1beta1\nkind: File\nName: 4"),
							},
							{
								Path:    "templates/deploy-4.yaml",
								Content: []byte("apiVersion: kots.io/v1beta1\nkind: File\nName: 5"),
							},
						},
						ErrorFiles: []BaseFile{},
						Bases: []Base{
							{
								Path: "crds",
								Files: []BaseFile{
									{
										Path:    "crd-2.yaml",
										Content: []byte("apiVersion: apiextensions.k8s.io/v1beta1\nkind: CustomResourceDefinition\nName: 6"),
									},
								},
								ErrorFiles: []BaseFile{},
								Bases:      []Base{},
							},
							{
								Path: "charts/my-sub-subchart-1",
								Files: []BaseFile{
									{
										Path:    "templates/deploy-5.yaml",
										Content: []byte("apiVersion: kots.io/v1beta1\nkind: File\nName: 7"),
									},
								},
								ErrorFiles: []BaseFile{},
								Bases:      []Base{},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderReplicatedHelmBase(tt.args.u, tt.args.renderOptions, tt.args.helmBase, tt.args.builder)
			if (err != nil) != tt.wantErr {
				t.Errorf("renderReplicatedHelmBase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("renderReplicatedHelmBase() \n\n%s", fmtJSONDiff(got, tt.want))
			}
		})
	}
}

func Test_extractHelmBases(t *testing.T) {
	type args struct {
		b Base
	}
	tests := []struct {
		name string
		args args
		want []Base
	}{
		{
			name: "recurse",
			args: args{
				b: Base{
					Path: "a",
					Bases: []Base{
						{
							Path: "b",
							Bases: []Base{
								{
									Path: "d",
								},
							},
						},
						{
							Path: "c",
						},
					},
				},
			},
			want: []Base{
				{
					Path: "a",
				},
				{
					Path: "a/b",
				},
				{
					Path: "a/b/d",
				},
				{
					Path: "a/c",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractHelmBases(tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractHelmBases() \n\n%s", fmtJSONDiff(got, tt.want))
			}
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
		expectedMultidoc   BaseFile
		expectedKotsKinds  map[string][]byte
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
              name: 'repl{{ ConfigOptionName "repl[[ .secretName ]]" }}'
              pod: repl{{ ConfigOption "podName" }}
              metaData:
                pod: repl{{ ConfigOption "podName"}}
                fileName: 'repl{{ ConfigOptionName "repl[[ .secretName ]]" }}'
              items:
                - key: "data"
                  path: '{{repl ConfigOptionName "repl[[ .secretName ]]" }}'
          - secret:
              name: "don't touch this either!"
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: multidoc-test
  annotations:
    key: val
  labels:
    app.kubernetes.io/name: multidoc
`),
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
  repl{{ ConfigOptionName "repl[[ .secretName ]]" }}: '{{repl ConfigOption "repl[[ .secretName ]]" }}'`),
					},
				},
			},
			renderOptions: &RenderOptions{
				Log: logger.NewCLILogger(ioutil.Discard),
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
					Path: "secret.yaml",
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
					Path: "secret-2.yaml",
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
					Path: "secret-3.yaml",
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
			expectedMultidoc: BaseFile{
				Path: "deployment-2.yaml",
				Content: []byte(
					`apiVersion: v1
kind: ServiceAccount
metadata:
  name: multidoc-test
  annotations:
    key: val
  labels:
    app.kubernetes.io/name: multidoc`),
			},
			expectedKotsKinds: map[string][]byte{
				"config.yaml": []byte(`apiVersion: kots.io/v1beta1
kind: Config
metadata:
  creationTimestamp: null
  name: config-sample
spec:
  groups:
  - description: info for pod
    items:
    - default: test
      name: podName
      type: text
      value: testPod
    - default: /var/www/html
      name: mountPath
      type: text
      value: ""
    name: podInfo
    title: ""
  - description: Buncha Secrets
    items:
    - countByGroup:
        secrets: 3
      default: onetwothree
      minimumCount: 1
      name: secretName
      repeatable: true
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
      title: Secret Name
      type: text
      value: ""
      valuesByGroup:
        secrets:
          secretName-1: MTIz
          secretName-2: MTIz
          secretName-3: MTIz
    name: secrets
    title: Secrets
status: {}
`,
				),
				"configvalues.yaml": []byte(`apiVersion: kots.io/v1beta1
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
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			base, _, kotsKinds, err := renderReplicated(test.upstream, test.renderOptions)
			req.NoError(err)

			decode := scheme.Codecs.UniversalDeserializer().Decode
			depobj, _, err := decode(test.expectedDeployment.Content, nil, nil)
			req.NoError(err)

			expectedDeployment := depobj.(*appsv1.Deployment)

			multidocobj, _, err := decode(test.expectedMultidoc.Content, nil, nil)
			req.NoError(err)

			expectedMultidoc := multidocobj.(*corev1.ServiceAccount)

			expKindsStruct, err := kotsutil.KotsKindsFromMap(test.expectedKotsKinds)
			req.NoError(err)
			kindsStruct, err := kotsutil.KotsKindsFromMap(kotsKinds)
			req.NoError(err)
			req.Equal(expKindsStruct, kindsStruct)

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

				if gvk.Kind == "ServiceAccount" {
					serviceAccount := obj.(*corev1.ServiceAccount)

					assert.Equal(t, expectedMultidoc, serviceAccount)
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

//go:embed testdata/postgresql-10.13.8.tgz
var postgresqlChart []byte

func Test_renderReplicatedHelm(t *testing.T) {

	setenv := func(content string) string {
		c, err := envsubst.Eval(content, util.TestGetenv)
		require.NoError(t, err)
		return c
	}
	tests := []struct {
		name              string
		upstream          *upstreamtypes.Upstream
		renderOptions     *RenderOptions
		expectedBase      Base
		expectedHelm      []Base
		expectedKotsKinds map[string][]byte
	}{
		{
			name: "basic test",
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
    podName:
      value: "testvalue"
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
`),
					},
				},
			},
			renderOptions: &RenderOptions{
				Log: logger.NewCLILogger(ioutil.Discard),
			},
			expectedBase: Base{Files: []BaseFile{
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
`),
				},
				{
					Path: "userdata/config.yaml",
					Content: []byte(`apiVersion: kots.io/v1beta1
kind: ConfigValues
metadata:
  name: test-app
spec:
  values:
    podName:
      value: "testvalue"
`),
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
        - name: testvalue
          image: httpd
`),
				},
			}},
			expectedHelm: []Base{},
			expectedKotsKinds: map[string][]byte{
				"config.yaml": []byte(`apiVersion: kots.io/v1beta1
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
      value: "testvalue"
`,
				),
				"configvalues.yaml": []byte(`apiVersion: kots.io/v1beta1
kind: ConfigValues
metadata:
  name: test-app
spec:
  values:
    podName:
      value: "testvalue"
`,
				),
			},
		},
		{
			name: "helm install test",
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
    podName:
      value: "testvalue"
`,
						),
					},
					{
						Path: "postgresql.yaml",
						Content: []byte(`
apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: postgresql
spec:
  # chart identifies a matching chart from a .tgz
  chart:
    name: postgresql
    chartVersion: 10.13.8
  helmVersion: v3
  useHelmInstall: true
  weight: 42
  values:
    postgresqlPassword: "abc123"
`),
					},
					{
						Path:    "postgresql-10.13.8.tgz",
						Content: postgresqlChart,
					},
				},
			},
			renderOptions: &RenderOptions{
				Log: logger.NewCLILogger(ioutil.Discard),
			},
			expectedBase: Base{Files: []BaseFile{
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
`),
				},
				{
					Path: "userdata/config.yaml",
					Content: []byte(`apiVersion: kots.io/v1beta1
kind: ConfigValues
metadata:
  name: test-app
spec:
  values:
    podName:
      value: "testvalue"
`),
				},
				{
					Path: "postgresql.yaml",
					Content: []byte(`
apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: postgresql
spec:
  # chart identifies a matching chart from a .tgz
  chart:
    name: postgresql
    chartVersion: 10.13.8
  helmVersion: v3
  useHelmInstall: true
  weight: 42
  values:
    postgresqlPassword: "abc123"
`),
				},
			}},
			expectedKotsKinds: map[string][]byte{
				"config.yaml": []byte(`apiVersion: kots.io/v1beta1
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
      value: "testvalue"
`,
				),
				"postgresql.yaml": []byte(`apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  creationTimestamp: null
  name: postgresql
spec:
  chart:
    name: postgresql
    chartVersion: 10.13.8
  helmVersion: v3
  useHelmInstall: true
  weight: 42
  exclude: ""
  values:
    postgresqlPassword: abc123
`,
				),
				"configvalues.yaml": []byte(`apiVersion: kots.io/v1beta1
kind: ConfigValues
metadata:
  name: test-app
spec:
  values:
    podName:
      value: "testvalue"
`,
				),
			},
			expectedHelm: []Base{
				{
					Path: "charts/postgresql",
					Files: []BaseFile{
						{
							Path: "templates/secrets.yaml",
							Content: []byte(setenv(`# Source: postgresql/templates/secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgresql
  labels:
    app.kubernetes.io/name: postgresql
    helm.sh/chart: postgresql-10.13.8
    app.kubernetes.io/instance: postgresql
    app.kubernetes.io/managed-by: Helm
  namespace: ${POD_NAMESPACE}
type: Opaque
data:
  postgresql-password: "YWJjMTIz"
`)),
						},
						{
							Path: "templates/statefulset.yaml",
							Content: []byte(setenv(`# Source: postgresql/templates/statefulset.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgresql-postgresql
  labels:
    app.kubernetes.io/name: postgresql
    helm.sh/chart: postgresql-10.13.8
    app.kubernetes.io/instance: postgresql
    app.kubernetes.io/managed-by: Helm
    app.kubernetes.io/component: primary
  annotations:
  namespace: ${POD_NAMESPACE}
spec:
  serviceName: postgresql-headless
  replicas: 1
  updateStrategy:
    type: RollingUpdate
  selector:
    matchLabels:
      app.kubernetes.io/name: postgresql
      app.kubernetes.io/instance: postgresql
      role: primary
  template:
    metadata:
      name: postgresql
      labels:
        app.kubernetes.io/name: postgresql
        helm.sh/chart: postgresql-10.13.8
        app.kubernetes.io/instance: postgresql
        app.kubernetes.io/managed-by: Helm
        role: primary
        app.kubernetes.io/component: primary
    spec:      
      affinity:
        podAffinity:
          
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - podAffinityTerm:
                labelSelector:
                  matchLabels:
                    app.kubernetes.io/name: postgresql
                    app.kubernetes.io/instance: postgresql
                    app.kubernetes.io/component: primary
                namespaces:
                  - "${POD_NAMESPACE}"
                topologyKey: kubernetes.io/hostname
              weight: 1
        nodeAffinity:
          
      securityContext:
        fsGroup: 1001
      automountServiceAccountToken: false
      containers:
        - name: postgresql
          image: docker.io/bitnami/postgresql:11.14.0-debian-10-r0
          imagePullPolicy: "IfNotPresent"
          resources:
            requests:
              cpu: 250m
              memory: 256Mi
          securityContext:
            runAsUser: 1001
          env:
            - name: BITNAMI_DEBUG
              value: "false"
            - name: POSTGRESQL_PORT_NUMBER
              value: "5432"
            - name: POSTGRESQL_VOLUME_DIR
              value: "/bitnami/postgresql"
            - name: PGDATA
              value: "/bitnami/postgresql/data"
            - name: POSTGRES_USER
              value: "postgres"
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgresql
                  key: postgresql-password
            - name: POSTGRESQL_ENABLE_LDAP
              value: "no"
            - name: POSTGRESQL_ENABLE_TLS
              value: "no"
            - name: POSTGRESQL_LOG_HOSTNAME
              value: "false"
            - name: POSTGRESQL_LOG_CONNECTIONS
              value: "false"
            - name: POSTGRESQL_LOG_DISCONNECTIONS
              value: "false"
            - name: POSTGRESQL_PGAUDIT_LOG_CATALOG
              value: "off"
            - name: POSTGRESQL_CLIENT_MIN_MESSAGES
              value: "error"
            - name: POSTGRESQL_SHARED_PRELOAD_LIBRARIES
              value: "pgaudit"
          ports:
            - name: tcp-postgresql
              containerPort: 5432
          livenessProbe:
            exec:
              command:
                - /bin/sh
                - -c
                - exec pg_isready -U "postgres" -h 127.0.0.1 -p 5432
            initialDelaySeconds: 30
            periodSeconds: 10
            timeoutSeconds: 5
            successThreshold: 1
            failureThreshold: 6
          readinessProbe:
            exec:
              command:
                - /bin/sh
                - -c
                - -e
                - |
                  exec pg_isready -U "postgres" -h 127.0.0.1 -p 5432
                  [ -f /opt/bitnami/postgresql/tmp/.initialized ] || [ -f /bitnami/postgresql/.initialized ]
            initialDelaySeconds: 5
            periodSeconds: 10
            timeoutSeconds: 5
            successThreshold: 1
            failureThreshold: 6
          volumeMounts:
            - name: dshm
              mountPath: /dev/shm
            - name: data
              mountPath: /bitnami/postgresql
              subPath: 
      volumes:
        - name: dshm
          emptyDir:
            medium: Memory
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes:
          - "ReadWriteOnce"
        resources:
          requests:
            storage: "8Gi"
`)),
						},
						{
							Path: "templates/svc-headless.yaml",
							Content: []byte(setenv(`# Source: postgresql/templates/svc-headless.yaml
apiVersion: v1
kind: Service
metadata:
  name: postgresql-headless
  labels:
    app.kubernetes.io/name: postgresql
    helm.sh/chart: postgresql-10.13.8
    app.kubernetes.io/instance: postgresql
    app.kubernetes.io/managed-by: Helm
    # Use this annotation in addition to the actual publishNotReadyAddresses
    # field below because the annotation will stop being respected soon but the
    # field is broken in some versions of Kubernetes:
    # https://github.com/kubernetes/kubernetes/issues/58662
    service.alpha.kubernetes.io/tolerate-unready-endpoints: "true"
  namespace: ${POD_NAMESPACE}
spec:
  type: ClusterIP
  clusterIP: None
  # We want all pods in the StatefulSet to have their addresses published for
  # the sake of the other Postgresql pods even before they're ready, since they
  # have to be able to talk to each other in order to become ready.
  publishNotReadyAddresses: true
  ports:
    - name: tcp-postgresql
      port: 5432
      targetPort: tcp-postgresql
  selector:
    app.kubernetes.io/name: postgresql
    app.kubernetes.io/instance: postgresql
`)),
						},
						{
							Path: "templates/svc.yaml",
							Content: []byte(setenv(`# Source: postgresql/templates/svc.yaml
apiVersion: v1
kind: Service
metadata:
  name: postgresql
  labels:
    app.kubernetes.io/name: postgresql
    helm.sh/chart: postgresql-10.13.8
    app.kubernetes.io/instance: postgresql
    app.kubernetes.io/managed-by: Helm
  annotations:
  namespace: ${POD_NAMESPACE}
spec:
  type: ClusterIP
  ports:
    - name: tcp-postgresql
      port: 5432
      targetPort: tcp-postgresql
  selector:
    app.kubernetes.io/name: postgresql
    app.kubernetes.io/instance: postgresql
    role: primary
`)),
						},
					},
					ErrorFiles: []BaseFile{},
					AdditionalFiles: []BaseFile{
						{
							Path: "values.yaml",
							Content: []byte(`audit:
  clientMinMessages: error
  logConnections: false
  logDisconnections: false
  logHostname: false
  logLinePrefix: ""
  logTimezone: ""
  pgAuditLog: ""
  pgAuditLogCatalog: "off"
common:
  exampleValue: common-chart
  global:
    imagePullSecrets: []
    imageRegistry: ""
    postgresql:
      existingSecret: ""
      postgresqlDatabase: ""
      postgresqlPassword: ""
      postgresqlUsername: ""
      replicationPassword: ""
      servicePort: ""
    storageClass: ""
commonAnnotations: {}
commonLabels: {}
configurationConfigMap: ""
containerSecurityContext:
  enabled: true
  runAsUser: 1001
customLivenessProbe: {}
customReadinessProbe: {}
customStartupProbe: {}
diagnosticMode:
  args:
  - infinity
  command:
  - sleep
  enabled: false
existingSecret: ""
extendedConfConfigMap: ""
extraDeploy: []
extraEnv: []
extraEnvVarsCM: ""
fullnameOverride: ""
global:
  imagePullSecrets: []
  imageRegistry: ""
  postgresql:
    existingSecret: ""
    postgresqlDatabase: ""
    postgresqlPassword: ""
    postgresqlUsername: ""
    replicationPassword: ""
    servicePort: ""
  storageClass: ""
image:
  debug: false
  pullPolicy: IfNotPresent
  pullSecrets: []
  registry: docker.io
  repository: bitnami/postgresql
  tag: 11.14.0-debian-10-r0
initdbPassword: ""
initdbScripts: {}
initdbScriptsConfigMap: ""
initdbScriptsSecret: ""
initdbUser: ""
ldap:
  baseDN: ""
  bind_password: ""
  bindDN: ""
  enabled: false
  port: ""
  prefix: ""
  scheme: ""
  search_attr: ""
  search_filter: ""
  server: ""
  suffix: ""
  tls: ""
  url: ""
lifecycleHooks: {}
livenessProbe:
  enabled: true
  failureThreshold: 6
  initialDelaySeconds: 30
  periodSeconds: 10
  successThreshold: 1
  timeoutSeconds: 5
metrics:
  customMetrics: {}
  enabled: false
  extraEnvVars: []
  image:
    pullPolicy: IfNotPresent
    pullSecrets: []
    registry: docker.io
    repository: bitnami/postgres-exporter
    tag: 0.10.0-debian-10-r116
  livenessProbe:
    enabled: true
    failureThreshold: 6
    initialDelaySeconds: 5
    periodSeconds: 10
    successThreshold: 1
    timeoutSeconds: 5
  prometheusRule:
    additionalLabels: {}
    enabled: false
    namespace: ""
    rules: []
  readinessProbe:
    enabled: true
    failureThreshold: 6
    initialDelaySeconds: 5
    periodSeconds: 10
    successThreshold: 1
    timeoutSeconds: 5
  resources: {}
  securityContext:
    enabled: false
    runAsUser: 1001
  service:
    annotations:
      prometheus.io/port: "9187"
      prometheus.io/scrape: "true"
    loadBalancerIP: ""
    type: ClusterIP
  serviceMonitor:
    additionalLabels: {}
    enabled: false
    interval: ""
    metricRelabelings: []
    namespace: ""
    relabelings: []
    scrapeTimeout: ""
nameOverride: ""
networkPolicy:
  allowExternal: true
  enabled: false
  explicitNamespacesSelector: {}
persistence:
  accessModes:
  - ReadWriteOnce
  annotations: {}
  enabled: true
  existingClaim: ""
  mountPath: /bitnami/postgresql
  selector: {}
  size: 8Gi
  storageClass: ""
  subPath: ""
pgHbaConfiguration: ""
postgresqlConfiguration: {}
postgresqlDataDir: /bitnami/postgresql/data
postgresqlDatabase: ""
postgresqlDbUserConnectionLimit: ""
postgresqlExtendedConf: {}
postgresqlInitdbArgs: ""
postgresqlInitdbWalDir: ""
postgresqlMaxConnections: ""
postgresqlPassword: abc123
postgresqlPghbaRemoveFilters: ""
postgresqlPostgresConnectionLimit: ""
postgresqlPostgresPassword: ""
postgresqlSharedPreloadLibraries: pgaudit
postgresqlStatementTimeout: ""
postgresqlTcpKeepalivesCount: ""
postgresqlTcpKeepalivesIdle: ""
postgresqlTcpKeepalivesInterval: ""
postgresqlUsername: postgres
primary:
  affinity: {}
  annotations: {}
  extraInitContainers: []
  extraPodSpec: {}
  extraVolumeMounts: []
  extraVolumes: []
  labels: {}
  nodeAffinityPreset:
    key: ""
    type: ""
    values: []
  nodeSelector: {}
  podAffinityPreset: ""
  podAnnotations: {}
  podAntiAffinityPreset: soft
  podLabels: {}
  priorityClassName: ""
  service:
    clusterIP: ""
    nodePort: ""
    type: ""
  sidecars: []
  tolerations: []
primaryAsStandBy:
  enabled: false
  primaryHost: ""
  primaryPort: ""
psp:
  create: false
rbac:
  create: false
readReplicas:
  affinity: {}
  annotations: {}
  extraInitContainers: []
  extraPodSpec: {}
  extraVolumeMounts: []
  extraVolumes: []
  labels: {}
  nodeAffinityPreset:
    key: ""
    type: ""
    values: []
  nodeSelector: {}
  persistence:
    enabled: true
  podAffinityPreset: ""
  podAnnotations: {}
  podAntiAffinityPreset: soft
  podLabels: {}
  priorityClassName: ""
  resources: {}
  service:
    clusterIP: ""
    nodePort: ""
    type: ""
  sidecars: []
  tolerations: []
  topologySpreadConstraints: []
readinessProbe:
  enabled: true
  failureThreshold: 6
  initialDelaySeconds: 5
  periodSeconds: 10
  successThreshold: 1
  timeoutSeconds: 5
replication:
  applicationName: my_application
  enabled: false
  numSynchronousReplicas: 0
  password: repl_password
  readReplicas: 1
  singleService: true
  synchronousCommit: "off"
  uniqueServices: false
  user: repl_user
resources:
  requests:
    cpu: 250m
    memory: 256Mi
schedulerName: ""
securityContext:
  enabled: true
  fsGroup: 1001
service:
  annotations: {}
  clusterIP: ""
  externalTrafficPolicy: Cluster
  loadBalancerIP: ""
  loadBalancerSourceRanges: []
  nodePort: ""
  port: 5432
  type: ClusterIP
serviceAccount:
  autoMount: false
  enabled: false
  name: ""
shmVolume:
  chmod:
    enabled: true
  enabled: true
  sizeLimit: ""
startupProbe:
  enabled: false
  failureThreshold: 10
  initialDelaySeconds: 30
  periodSeconds: 15
  successThreshold: 1
  timeoutSeconds: 5
terminationGracePeriodSeconds: ""
tls:
  autoGenerated: false
  certCAFilename: ""
  certFilename: ""
  certKeyFilename: ""
  certificatesSecret: ""
  crlFilename: ""
  enabled: false
  preferServerCiphers: true
updateStrategy:
  type: RollingUpdate
usePasswordFile: false
volumePermissions:
  enabled: false
  image:
    pullPolicy: IfNotPresent
    pullSecrets: []
    registry: docker.io
    repository: bitnami/bitnami-shell
    tag: 10-debian-10-r248
  securityContext:
    runAsUser: 0
`),
						},
						{
							Path: "Chart.yaml",
							Content: []byte(`annotations:
  category: Database
apiVersion: v2
appVersion: 11.14.0
dependencies:
- name: common
  repository: https://charts.bitnami.com/bitnami
  version: 1.x.x
description: Chart for PostgreSQL, an object-relational database management system
  (ORDBMS) with an emphasis on extensibility and on standards-compliance.
home: https://github.com/bitnami/charts/tree/master/bitnami/postgresql
icon: https://bitnami.com/assets/stacks/postgresql/img/postgresql-stack-220x234.png
keywords:
- postgresql
- postgres
- database
- sql
- replication
- cluster
maintainers:
- email: containers@bitnami.com
  name: Bitnami
- email: cedric@desaintmartin.fr
  name: desaintmartin
name: postgresql
sources:
- https://github.com/bitnami/bitnami-docker-postgresql
- https://www.postgresql.org/
version: 10.13.8
`),
						},
						{
							Path: "Chart.lock",
							Content: []byte(`dependencies:
- name: common
  repository: https://charts.bitnami.com/bitnami
  version: 1.10.1
digest: sha256:84f150f2d532eb5cb38ad0201fc071d7a1c43d1e815330cd8dedd5bc268575ec
generated: "2021-10-29T07:02:39.999761537Z"
`),
						},
					},
				},
				{
					Path:       "charts/postgresql/charts/common",
					Files:      []BaseFile{},
					ErrorFiles: []BaseFile{},
					AdditionalFiles: []BaseFile{
						{
							Path: "Chart.yaml",
							Content: []byte(`annotations:
  category: Infrastructure
apiVersion: v2
appVersion: 1.10.0
description: A Library Helm Chart for grouping common logic between bitnami charts.
  This chart is not deployable by itself.
home: https://github.com/bitnami/charts/tree/master/bitnami/common
icon: https://bitnami.com/downloads/logos/bitnami-mark.png
keywords:
- common
- helper
- template
- function
- bitnami
maintainers:
- email: containers@bitnami.com
  name: Bitnami
name: common
sources:
- https://github.com/bitnami/charts
- http://www.bitnami.com/
type: library
version: 1.10.1
`),
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			base, helmBase, kotsKinds, err := renderReplicated(test.upstream, test.renderOptions)
			req.NoError(err)
			req.ElementsMatch(test.expectedHelm, helmBase)
			req.ElementsMatch(test.expectedBase.Files, base.Files)

			expKindsStruct, err := kotsutil.KotsKindsFromMap(test.expectedKotsKinds)
			req.NoError(err)
			kindsStruct, err := kotsutil.KotsKindsFromMap(kotsKinds)
			req.NoError(err)
			req.Equal(expKindsStruct, kindsStruct)
		})
	}
}
