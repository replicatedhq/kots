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
		wantSubCharts map[string][]BaseFile
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
			wantSubCharts: map[string][]BaseFile{
				"my-subchart-1": {
					{Path: "templates/deploy-2.yaml", Content: []byte("file: 3")},
				},
				"my-subchart-2": {
					{Path: "templates/deploy-3.yaml", Content: []byte("file: 4")},
					{Path: "templates/deploy-4.yaml", Content: []byte("file: 5")},
					{Path: "crds/crd-2.yaml", Content: []byte("file: 6")},
					{Path: "charts/my-sub-subchart-1/templates/deploy-5.yaml", Content: []byte("file: 7")},
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
