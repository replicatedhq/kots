package base

import (
	"reflect"
	"testing"

	"github.com/replicatedhq/kots/pkg/template"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
