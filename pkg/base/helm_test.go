package base

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/pmezard/go-difflib/difflib"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
)

func Test_checkChartForVersion(t *testing.T) {
	v3Test := map[string]interface{}{
		"apiVersion": "v2",
		"name":       "testChart",
		"type":       "application",
		"version":    "v0.0.1",
		"appVersion": "v1.0.0",
	}

	v2Test := map[string]interface{}{
		"name":       "testChart",
		"type":       "application",
		"version":    "v2",
		"appVersion": "v2",
	}

	tests := []struct {
		name    string
		chart   map[string]interface{}
		want    string
		wantErr bool
	}{
		{
			name:    "version 3 API",
			chart:   v3Test,
			want:    "v3",
			wantErr: false,
		},
		{
			name:    "version 2",
			chart:   v2Test,
			want:    "v2",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlContent, err := yaml.Marshal(tt.chart)
			if err != nil {
				t.Errorf("checkChartForVersion() error = %v", err)
			}
			chartFile := upstreamtypes.UpstreamFile{
				Content: yamlContent,
			}
			got, err := checkChartForVersion(&chartFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkChartForVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkChartForVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_RenderHelm(t *testing.T) {
	type args struct {
		upstream      *upstreamtypes.Upstream
		renderOptions *RenderOptions
	}
	tests := []struct {
		name    string
		args    args
		want    *Base
		wantErr bool
	}{
		{
			name: "helm v3 namespace insertion",
			args: args{
				upstream: &upstreamtypes.Upstream{
					Name: "namespace-test",
					Files: []upstreamtypes.UpstreamFile{
						{
							Path:    "Chart.yaml",
							Content: []byte("name: test-chart\nversion: 0.1.0"),
						},
						{
							Path:    "templates/deploy-1.yaml",
							Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-1\n  namespace: test-one"),
						},
						{
							Path:    "templates/deploy-2.yaml",
							Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2"),
						},
					},
				},
				renderOptions: &RenderOptions{
					HelmVersion: "v3",
					Namespace:   "test-two",
				},
			},
			want: &Base{
				Files: []BaseFile{
					{
						Path:    "chartHelmSecret.yaml",
						Content: []byte("apiVersion: v1\ndata:\n  release: SDRzSUFBQUFBQUFDLzdTVFgydmJNQlRGdjRxNWU1VVQyeU9EQ3Zvd3Qyc29iQWx0V2V4NEhrT1JsVVNML21ISlRrM3hkeDlLNHFZdzhqQlluMndkWDg3OWNYVDhBb3BJQnZqd3NJWlFGanBtSFNEZ2FxMEJ2OENhMTliOXFwZ1J1bU1WWUFBRWd2d2xWVXd3ZHo1WVduUGp1RmFBNGJidWdycFJBZFhTK0NGQVlCMXhqUVVNcnlZOUFyb2x0Zk1ySlhPa0lvNzQ5eE9laHdxUEF3aGFWdHVqZFRTS1J4RWdJSVl2WHNVMjltNUMweDFnMVFpQndERnBCSEhNQXY3eHh2SWtqbzhRWVR6cWlCUWUvN0FibG5scUZuTFIwVVMwcTk5NnM4elRQZTJ1a204UHBpYlpaRGZuNlpjaVQrM3FvM0JGTm9sdVpDeXE2ZDF1bVQ5dTV4dTl1WjlPdHF2cys2Zjc2YU9nMCtlV1BVWFBOL3p6WmlYdlhKSFA5c3RzSnVZOGpZcDhGbjNOcnByaTRmb2FlblFaTUhsL3dBNzZud2hhSWhvZjFqRTlTN2RNa3VHMDVtTDQxQ09RUlBHMUx3eUdNQXhMOVNGNDBrMU5HUTdPTnphK2tIU3B6dGVHQTJLTUhiZHhxWFpjVlRpNFBVeEtwbHlwaGo3Z1VnV0JEd2NIZzlHZ0hLcDdXcW9WSzlVLzBDVC9peVlwMWR0Mnh1ajhWdzBWZG5zTi9aOEFBQUQvLzRkS0VXOTFBd0FB\nkind: Secret\nmetadata:\n  creationTimestamp: null\n  labels:\n    createdAt: \"1\"\n    name: namespace-test\n    owner: helm\n    status: deployed\n    version: \"1\"\n  name: sh.helm.release.v1.namespace-test.v1\n  namespace: test-two\ntype: helm.sh/release.v1"),
					},
					{
						Path: "deploy-1.yaml",
						Content: []byte("# Source: test-chart/templates/deploy-1.yaml\napiVersion: apps/v1\nkind: Deployment\n" +
							"metadata:\n  name: deploy-1\n  namespace: test-one"),
					},
					{
						Path:    "deploy-2.yaml",
						Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2\n  namespace: test-two"),
					},
				},
				AdditionalFiles: []BaseFile{
					{
						Path:    "Chart.yaml",
						Content: []byte("name: test-chart\nversion: 0.1.0"),
					},
				},
			},
		},
		{
			name: "helm v2 namespace insertion",
			args: args{
				upstream: &upstreamtypes.Upstream{
					Name: "namespace-test",
					Files: []upstreamtypes.UpstreamFile{
						{
							Path:    "Chart.yaml",
							Content: []byte("name: test-chart\nversion: 0.1.0"),
						},
						{
							Path:    "templates/deploy-2.yaml",
							Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2"),
						},
					},
				},
				renderOptions: &RenderOptions{
					HelmVersion: "v2",
					Namespace:   "test-two",
				},
			},
			want: &Base{
				Files: []BaseFile{
					{
						Path:    "deploy-2.yaml",
						Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2\n  namespace: test-two"),
					},
				},
				AdditionalFiles: []BaseFile{
					{
						Path:    "Chart.yaml",
						Content: []byte("name: test-chart\nversion: 0.1.0"),
					},
				},
			},
		},
		{
			name: "helm v3 namespace insertion with multidoc",
			args: args{
				upstream: &upstreamtypes.Upstream{
					Name: "namespace-test",
					Files: []upstreamtypes.UpstreamFile{
						{
							Path:    "Chart.yaml",
							Content: []byte("name: test-chart\nversion: 0.1.0"),
						},
						{
							Path: "templates/deploy.yaml",
							Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-1\n  namespace: test-one\n" +
								"---\n" +
								"apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2\n  namespace: test-two"),
						},
					},
				},
				renderOptions: &RenderOptions{
					HelmVersion: "v3",
					Namespace:   "test-two",
				},
			},
			want: &Base{
				Files: []BaseFile{
					{
						Path:    "chartHelmSecret.yaml",
						Content: []byte("apiVersion: v1\ndata:\n  release: SDRzSUFBQUFBQUFDLzd4VFhXdmJNQlQ5SytidVZVN3NsQXdxNk1QY3JxR3NTMmpDWXNmekdMS3NKRnIwaFNYYk5TWC9mVGlPbThLNngrMUp1a2VYY3c3M0hyMkFJcElCUGgzV0VNcDh4NndEQkZ4dE5lQVgyUExTdXA4Rk0wSzNyQUFNZ0VDUVA2Q0NDZVl1aGFVbE40NXJCUmp1eXRZcksrVlJMVTNYQkFpc0k2NnlnT0dWNUlpQTdrbnBPa25KSENtSUk5MzliSzh6NWZjTkNHcFcycDQ2R0lXakFCQVF3OWV2WUIxMmJFTFRBMkJWQ1lIQU1Xa0VjY3dDL3Y2RzhneU9leE9qbGtqUm1UOHB3eWFKekZxdVd6b1JkZjVMN3paSjFORDJldkwxeVpRa25oNFdQUHFjSnBITnI0Ukw0Mmx3SzBOUnpPNFBtMlM1WCt6MDdtRTIzZWZ4dDQ4UHM2V2dzK2VhcllMblcvNXBsOHQ3bHlielpoUFB4WUpIUVpyTWc4ZjR1a3FmdEh0Y0JWLytnVzc3Vjkxa2VaVTNOemR3L0lHZ0pxTHFKdFNQek5JOWsyU290bHdNVDBjRWtpaSs3VktDd2ZmOVRIM3dWcm9xS2NQZVpVM2pkOGVicWN1bXNFZU1zZU02ek5TQnF3SjdkNmMreVpUTDFCQUJuQ25QNnhhR3ZaN0dEd2ZrbE5henBGWXNVLy9meStROUw2N1JtWHFiMGhCZGZ0Y1FaZGRvT1A0T0FBRC8vd0RtQjhkOUF3QUE=\nkind: Secret\nmetadata:\n  creationTimestamp: null\n  labels:\n    createdAt: \"1\"\n    name: namespace-test\n    owner: helm\n    status: deployed\n    version: \"1\"\n  name: sh.helm.release.v1.namespace-test.v1\n  namespace: test-two\ntype: helm.sh/release.v1"),
					},
					{
						Path: "deploy.yaml",
						Content: []byte("# Source: test-chart/templates/deploy.yaml\n" +
							"apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-1\n  namespace: test-one\n" +
							"---\n" +
							"# Source: test-chart/templates/deploy.yaml\n" +
							"apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2\n  namespace: test-two"),
					},
				},
				AdditionalFiles: []BaseFile{
					{
						Path:    "Chart.yaml",
						Content: []byte("name: test-chart\nversion: 0.1.0"),
					},
				},
			},
		},
		{
			name: "helm v2 namespace insertion with multidoc",
			args: args{
				upstream: &upstreamtypes.Upstream{
					Name: "namespace-test",
					Files: []upstreamtypes.UpstreamFile{
						{
							Path:    "Chart.yaml",
							Content: []byte("name: test-chart\nversion: 0.1.0"),
						},
						{
							Path: "templates/deploy.yaml",
							Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-1\n  namespace: test-one\n" +
								"---\n" +
								"apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2\n  namespace: test-two"),
						},
					},
				},
				renderOptions: &RenderOptions{
					HelmVersion: "v2",
					Namespace:   "test-two",
				},
			},
			want: &Base{
				Files: []BaseFile{
					{
						Path: "deploy.yaml",
						Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-1\n  namespace: test-one\n" +
							"---\n" +
							"apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2\n  namespace: test-two"),
					},
				},
				AdditionalFiles: []BaseFile{
					{
						Path:    "Chart.yaml",
						Content: []byte("name: test-chart\nversion: 0.1.0"),
					},
				},
			},
		},
		{
			name: "namespace insertion with invalid yaml",
			args: args{
				upstream: &upstreamtypes.Upstream{
					Name: "namespace-test",
					Files: []upstreamtypes.UpstreamFile{
						{
							Path:    "Chart.yaml",
							Content: []byte("name: test-chart\nversion: 0.1.0"),
						},
						{
							Path:    "templates/invalid.yaml",
							Content: []byte(" invalid\n\nyaml"),
						},
					},
				},
				renderOptions: &RenderOptions{
					HelmVersion: "v2",
					Namespace:   "test-two",
				},
			},
			want: &Base{
				Files: []BaseFile{
					{
						Path:    "invalid.yaml",
						Content: []byte("invalid\n\nyaml"),
					},
				},
				AdditionalFiles: []BaseFile{
					{
						Path:    "Chart.yaml",
						Content: []byte("name: test-chart\nversion: 0.1.0"),
					},
				},
			},
		},
		{
			name: "namespace insertion with cluster scoped resources",
			args: args{
				upstream: &upstreamtypes.Upstream{
					Name: "namespace-test",
					Files: []upstreamtypes.UpstreamFile{
						{
							Path:    "Chart.yaml",
							Content: []byte("name: test-chart\nversion: 0.1.0"),
						},
						{
							Path:    "templates/crd.yaml",
							Content: []byte("apiVersion: v1\nkind: CustomResourceDefinition\nmetadata:\n  name: example-crd\nspec:\n  scope: Cluster"),
						},
					},
				},
				renderOptions: &RenderOptions{
					HelmVersion: "v3",
					Namespace:   "test-two",
				},
			},
			want: &Base{
				Files: []BaseFile{
					{
						Path:    "chartHelmSecret.yaml",
						Content: []byte("apiVersion: v1\ndata:\n  release: SDRzSUFBQUFBQUFDLzJTU1cyL2FNQlRIdjRwMTl1cTBCRGFrV3RyREJpeGkwNWpXaVhCWnBzbllKcmo0SnRzSlN5dSsrMlFvcFZLZmtuUFIvL3pQK2ZrSkROVUN5T2tUSEdVaWl5SkV3Q0ROMWdKNWdxMzBJZjdsd2luYkNRNEVBSU9pYjFKY0tCR3ZRV0JldWlpdEFRSmozeUhmR01Tc2Rxa0pNSVJJWXhPQXdJdklFUVBiVVIvVFNDMGk1VFRTOVA5c0w1bkt6ZzBZV3VIRFdicDNrOS8wQUFOMXNueEp0bmxTVTVidGdaaEdLUXhSYUtkb0ZBSEk3MWVTejhsYjV2bE5SN1ZLems5alliWDg3RXBkZHF5djJzMkRyZm5ENUJ2dHEyWTl0dlhQUWZuSWk3czQxK1hqWmxCMnEzNDVXUy9XYnFOVmp5N3VtcEhPRlMrKzdGZkwrOTJQMnRiVDRzTnVzNWdQcDBYNWZyWElENXRpSGxlRHIvdVJtUjNXaSsvRGtmeFVzLzZzWmNWOE9KM01BbC9PZXV2bDlDTWMvMkJvcVdxUzdmTWVnZTJFcHBkb0s5V2xkTVNncVpIYmhJNUFsbVdWZVlkKzJjWXpRZEQxZHJkdmQ2N005WFlFdFhsbDl0Sndna1pOaUZiZmkzQVNHWXV0TkRMeHJNeUZEcWtNUXVtV0JJbC9OS0hObU9lVkNVNndVeTB3NndSQkk5V0VLSHhsWHBQTDhmWEZYZkRHZzRYai93QUFBUC8vSVVPakpaRUNBQUE9\nkind: Secret\nmetadata:\n  creationTimestamp: null\n  labels:\n    createdAt: \"1\"\n    name: namespace-test\n    owner: helm\n    status: deployed\n    version: \"1\"\n  name: sh.helm.release.v1.namespace-test.v1\n  namespace: test-two\ntype: helm.sh/release.v1"),
					},
					{
						Path:    "crd.yaml",
						Content: []byte("# Source: test-chart/templates/crd.yaml\napiVersion: v1\nkind: CustomResourceDefinition\nmetadata:\n  name: example-crd\nspec:\n  scope: Cluster"),
					},
				},
				AdditionalFiles: []BaseFile{
					{
						Path:    "Chart.yaml",
						Content: []byte("name: test-chart\nversion: 0.1.0"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderHelm(tt.args.upstream, tt.args.renderOptions)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderHelm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RenderHelm() \n\n%s", fmtJSONDiff(got, tt.want))
			}
		})
	}
}

func Test_writeHelmBase(t *testing.T) {
	type args struct {
		chartName     string
		baseFiles     []BaseFile
		renderOptions *RenderOptions
	}
	tests := []struct {
		name    string
		args    args
		want    *Base
		wantErr bool
	}{
		{
			name: "test split",
			args: args{
				chartName: "my-chart",
				baseFiles: []BaseFile{
					{Path: "multi.yaml", Content: []byte("a: a\n---\nb: b")},
				},
				renderOptions: &RenderOptions{
					SplitMultiDocYAML: true,
				},
			},
			want: &Base{
				Path: "charts/my-chart",
				Files: []BaseFile{
					{
						Path:    "multi-1.yaml",
						Content: []byte("a: a"),
					},
					{
						Path:    "multi-2.yaml",
						Content: []byte("b: b"),
					},
				},
			},
		},
		{
			name: "test no split",
			args: args{
				chartName: "my-chart",
				baseFiles: []BaseFile{
					{Path: "multi.yaml", Content: []byte("a: a\n---\nb: b")},
				},
				renderOptions: &RenderOptions{},
			},
			want: &Base{
				Path: "charts/my-chart",
				Files: []BaseFile{
					{
						Path:    "multi.yaml",
						Content: []byte("a: a\n---\nb: b"),
					},
				},
			},
		},
		{
			name: "test crds and subcharts",
			args: args{
				chartName: "my-chart",
				baseFiles: []BaseFile{
					{Path: "templates/deploy-1.yaml", Content: []byte("file: 1")},
					{Path: "crds/crd-1.yaml", Content: []byte("file: 2")},
					{Path: "charts/my-subchart-1/templates/deploy-2.yaml", Content: []byte("file: 3")},
					{Path: "charts/my-subchart-2/templates/deploy-3.yaml", Content: []byte("file: 4")},
					{Path: "charts/my-subchart-2/templates/deploy-4.yaml", Content: []byte("file: 5")},
					{Path: "charts/my-subchart-2/crds/crd-2.yaml", Content: []byte("file: 6")},
					{Path: "charts/my-subchart-2/charts/my-sub-subchart-1/templates/deploy-5.yaml", Content: []byte("file: 7")},
				},
				renderOptions: &RenderOptions{},
			},
			want: &Base{
				Path: "charts/my-chart",
				Files: []BaseFile{
					{
						Path:    "templates/deploy-1.yaml",
						Content: []byte("file: 1"),
					},
				},
				Bases: []Base{
					{
						Path: "crds",
						Files: []BaseFile{
							{
								Path:    "crd-1.yaml",
								Content: []byte("file: 2"),
							},
						},
					},
					{
						Path: "charts/my-subchart-1",
						Files: []BaseFile{
							{
								Path:    "templates/deploy-2.yaml",
								Content: []byte("file: 3"),
							},
						},
					},
					{
						Path: "charts/my-subchart-2",
						Files: []BaseFile{
							{
								Path:    "templates/deploy-3.yaml",
								Content: []byte("file: 4"),
							},
							{
								Path:    "templates/deploy-4.yaml",
								Content: []byte("file: 5"),
							},
						},
						Bases: []Base{
							{
								Path: "crds",
								Files: []BaseFile{
									{
										Path:    "crd-2.yaml",
										Content: []byte("file: 6"),
									},
								},
							},
							{
								Path: "charts/my-sub-subchart-1",
								Files: []BaseFile{
									{
										Path:    "templates/deploy-5.yaml",
										Content: []byte("file: 7"),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := writeHelmBase(tt.args.chartName, tt.args.baseFiles, tt.args.renderOptions)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeHelmBase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("writeHelmBase() \n\n%s", fmtJSONDiff(got, tt.want))
			}
		})
	}
}

func Test_splitHelmFiles(t *testing.T) {
	type args struct {
		baseFiles []BaseFile
	}
	tests := []struct {
		name          string
		args          args
		wantRest      []BaseFile
		wantCrds      []BaseFile
		wantSubCharts []subChartBase
	}{
		{
			name: "basic",
			args: args{
				baseFiles: []BaseFile{
					{Path: "templates/deploy-1.yaml", Content: []byte("file: 1")},
					{Path: "crds/crd-1.yaml", Content: []byte("file: 2")},
					{Path: "charts/my-subchart-1/templates/deploy-2.yaml", Content: []byte("file: 3")},
					{Path: "charts/my-subchart-2/templates/deploy-3.yaml", Content: []byte("file: 4")},
					{Path: "charts/my-subchart-2/templates/deploy-4.yaml", Content: []byte("file: 5")},
					{Path: "charts/my-subchart-2/crds/crd-2.yaml", Content: []byte("file: 6")},
					{Path: "charts/my-subchart-2/charts/my-sub-subchart-1/templates/deploy-5.yaml", Content: []byte("file: 7")},
				},
			},
			wantRest: []BaseFile{
				{Path: "templates/deploy-1.yaml", Content: []byte("file: 1")},
			},
			wantCrds: []BaseFile{
				{Path: "crd-1.yaml", Content: []byte("file: 2")},
			},
			wantSubCharts: []subChartBase{
				{
					Name: "my-subchart-1",
					BaseFiles: []BaseFile{
						{Path: "templates/deploy-2.yaml", Content: []byte("file: 3")},
					},
				},
				{
					Name: "my-subchart-2",
					BaseFiles: []BaseFile{
						{Path: "templates/deploy-3.yaml", Content: []byte("file: 4")},
						{Path: "templates/deploy-4.yaml", Content: []byte("file: 5")},
						{Path: "crds/crd-2.yaml", Content: []byte("file: 6")},
						{Path: "charts/my-sub-subchart-1/templates/deploy-5.yaml", Content: []byte("file: 7")},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRest, gotCrds, gotSubCharts := splitHelmFiles(tt.args.baseFiles)
			if !reflect.DeepEqual(gotRest, tt.wantRest) {
				t.Errorf("splitHelmFiles() rest \n\n%s", fmtJSONDiff(gotRest, tt.wantRest))
			}
			if !reflect.DeepEqual(gotCrds, tt.wantCrds) {
				t.Errorf("splitHelmFiles() crds \n\n%s", fmtJSONDiff(gotCrds, tt.wantCrds))
			}
			if !reflect.DeepEqual(gotSubCharts, tt.wantSubCharts) {
				t.Errorf("splitHelmFiles() subCharts \n\n%s", fmtJSONDiff(gotSubCharts, tt.wantSubCharts))
			}
		})
	}
}

func Test_writeHelmBaseFile(t *testing.T) {
	type args struct {
		baseFile      BaseFile
		renderOptions *RenderOptions
	}
	tests := []struct {
		name    string
		args    args
		want    []BaseFile
		wantErr bool
	}{
		{
			name: "split",
			args: args{
				baseFile: BaseFile{
					Path:    "multi.yaml",
					Content: []byte("a: a\n---\nb: b"),
				},
				renderOptions: &RenderOptions{SplitMultiDocYAML: true},
			},
			want: []BaseFile{
				{
					Path:    "multi-1.yaml",
					Content: []byte("a: a"),
				},
				{
					Path:    "multi-2.yaml",
					Content: []byte("b: b"),
				},
			},
		},
		{
			name: "no split",
			args: args{
				baseFile: BaseFile{
					Path:    "multi.yaml",
					Content: []byte("a: a\n---\nb: b"),
				},
				renderOptions: &RenderOptions{},
			},
			want: []BaseFile{
				{
					Path:    "multi.yaml",
					Content: []byte("a: a\n---\nb: b"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := writeHelmBaseFile(tt.args.baseFile, tt.args.renderOptions)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeHelmBaseFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("writeHelmBaseFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_removeCommonPrefix(t *testing.T) {
	type args struct {
		baseFiles []BaseFile
	}
	tests := []struct {
		name string
		args args
		want []BaseFile
	}{
		{
			name: "basic",
			args: args{
				baseFiles: []BaseFile{
					{Path: "a/b/c/d"},
					{Path: "a/b/c/e"},
					{Path: "a/b/d/e"},
				},
			},
			want: []BaseFile{
				{Path: "c/d"},
				{Path: "c/e"},
				{Path: "d/e"},
			},
		},
		{
			name: "one file",
			args: args{
				baseFiles: []BaseFile{
					{Path: "a/b/c"},
				},
			},
			want: []BaseFile{
				{Path: "c"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := removeCommonPrefix(tt.args.baseFiles); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("removeCommonPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func fmtJSONDiff(got, want interface{}) string {
	a, _ := json.MarshalIndent(got, "", "  ")
	b, _ := json.MarshalIndent(want, "", "  ")
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(a)),
		B:        difflib.SplitLines(string(b)),
		FromFile: "Got",
		ToFile:   "Want",
		Context:  1,
	}
	diffStr, _ := difflib.GetUnifiedDiffString(diff)
	return fmt.Sprintf("got:\n%s \n\nwant:\n%s \n\ndiff:\n%s", got, want, diffStr)
}
