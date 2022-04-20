package client

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_getLabelSelector(t *testing.T) {
	tests := []struct {
		name             string
		appLabelSelector metav1.LabelSelector
		want             string
	}{
		{
			name: "no requirements",
			appLabelSelector: metav1.LabelSelector{
				MatchLabels:      nil,
				MatchExpressions: nil,
			},
			want: "",
		},
		{
			name: "one requirement",
			appLabelSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kots.io/label": "abc",
				},
				MatchExpressions: nil,
			},
			want: "kots.io/label=abc",
		},
		{
			name: "two requirements",
			appLabelSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kots.io/label": "abc",
					"otherlabel":    "xyz",
				},
				MatchExpressions: nil,
			},
			want: "kots.io/label=abc,otherlabel=xyz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, getLabelSelector(&tt.appLabelSelector))
		})
	}
}

func Test_getSortedCharts(t *testing.T) {
	type file struct {
		path     string
		contents string
	}
	tests := []struct {
		name       string
		files      []file
		kotsCharts []*v1beta1.HelmChart
		want       []orderedDir
	}{
		{
			name: "chart without an entry in kotsCharts should work", // this should not come up in practice but is good to reduce risk
			files: []file{
				{
					path:     "chart1/Chart.yaml",
					contents: `name: chart1name`,
				},
			},
			want: []orderedDir{
				{
					Dir:       "chart1",
					ChartName: "chart1name",
				},
			},
		},
		{
			name: "four charts, one not weighted, two with equal weights, one irrelevant file",
			files: []file{
				{
					path:     "chart1/irrelevant", // this file should be ignored
					contents: "abc123",
				},
				{
					path: "chart1/Chart.yaml",
					contents: `
name: chart1
version: "ver1"
`,
				},
				{
					path: "chart2/Chart.yaml",
					contents: `
name: chart2
version: "v1"
`,
				},
				{
					path: "chart3/Chart.yaml",
					contents: `
name: chart3
version: "v1"
`,
				},
				{
					path: "chart4/Chart.yaml",
					contents: `
name: chart4
version: "v1"
`,
				},
			},
			kotsCharts: []*v1beta1.HelmChart{
				{
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "chart1",
							ChartVersion: "ver1",
						},
						Weight: 1,
					},
				},
				{
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "chart2",
							ChartVersion: "v1",
						},
						Weight: 1,
					},
				},
				{
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "chart3",
							ChartVersion: "v1",
						},
						Weight: 5,
					},
				},
				{
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "chart4",
							ChartVersion: "v1",
						},
					},
				},
			},
			want: []orderedDir{
				{
					Dir:          "chart4",
					ChartName:    "chart4",
					ChartVersion: "v1",
				},
				{
					Dir:          "chart1",
					Weight:       1,
					ChartName:    "chart1",
					ChartVersion: "ver1",
				},
				{
					Dir:          "chart2",
					Weight:       1,
					ChartName:    "chart2",
					ChartVersion: "v1",
				},
				{
					Dir:          "chart3",
					Weight:       5,
					ChartName:    "chart3",
					ChartVersion: "v1",
				},
			},
		},
		{
			name: "negative weights before no weight",
			files: []file{
				{
					path: "chart1/Chart.yaml",
					contents: `
name: c1
version: ver1
`,
				},
				{
					path: "chart2/Chart.yaml",
					contents: `
name: c2
`,
				},
			},
			kotsCharts: []*v1beta1.HelmChart{
				{
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "c1",
							ChartVersion: "ver1",
						},
						Weight: -5,
					},
				},
			},
			want: []orderedDir{
				{
					Dir:          "chart1",
					Weight:       -5,
					ChartName:    "c1",
					ChartVersion: "ver1",
				},
				{
					Dir:       "chart2",
					ChartName: "c2",
				},
			},
		},
		{
			name: "same name, different versions",
			files: []file{
				{
					path: "chart1/Chart.yaml",
					contents: `
name: generic
version: ver1
`,
				},
				{
					path: "chart2/Chart.yaml",
					contents: `
name: generic
version: ver2
`,
				},
			},
			kotsCharts: []*v1beta1.HelmChart{
				{
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "generic",
							ChartVersion: "ver1",
						},
						Weight: -1,
					},
				},
				{
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "generic",
							ChartVersion: "ver2",
						},
						Weight: 2,
					},
				},
			},
			want: []orderedDir{
				{
					Dir:          "chart1",
					ChartName:    "generic",
					ChartVersion: "ver1",
					Weight:       -1,
				},
				{
					Dir:          "chart2",
					Weight:       2,
					ChartName:    "generic",
					ChartVersion: "ver2",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			tempdir, err := ioutil.TempDir("", "kots_getSortedCharts")
			req.NoError(err)
			defer os.RemoveAll(tempdir)

			// populate host directory
			for _, file := range tt.files {
				err = os.MkdirAll(filepath.Dir(filepath.Join(tempdir, file.path)), os.ModePerm)
				req.NoError(err)

				err = ioutil.WriteFile(filepath.Join(tempdir, file.path), []byte(file.contents), os.ModePerm)
				req.NoError(err)
			}

			got, err := getSortedCharts(tempdir, tt.kotsCharts)
			req.NoError(err)
			req.Equal(tt.want, got)
		})
	}
}
