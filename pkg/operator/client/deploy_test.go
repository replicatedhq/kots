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

func Test_getSortedCharts(t *testing.T) {
	type file struct {
		path     string
		contents string
	}
	tests := []struct {
		name            string
		files           []file
		kotsCharts      []v1beta1.HelmChart
		targetNamespace string
		isUninstall     bool
		want            []orderedDir
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
					Name:        "chart1",
					ChartName:   "chart1name",
					ReleaseName: "chart1name",
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
			kotsCharts: []v1beta1.HelmChart{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "chart1",
					},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "chart1",
							ChartVersion: "ver1",
						},
						Weight: 1,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "chart2",
					},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "chart2",
							ChartVersion: "v1",
						},
						Weight: 1,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "chart3",
					},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "chart3",
							ChartVersion: "v1",
						},
						Weight: 5,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "chart4",
					},
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
					Name:         "chart4",
					ChartName:    "chart4",
					ChartVersion: "v1",
					ReleaseName:  "chart4",
				},
				{
					Name:         "chart1",
					Weight:       1,
					ChartName:    "chart1",
					ChartVersion: "ver1",
					ReleaseName:  "chart1",
				},
				{
					Name:         "chart2",
					Weight:       1,
					ChartName:    "chart2",
					ChartVersion: "v1",
					ReleaseName:  "chart2",
				},
				{
					Name:         "chart3",
					Weight:       5,
					ChartName:    "chart3",
					ChartVersion: "v1",
					ReleaseName:  "chart3",
				},
			},
		},
		{
			name: "four charts, one not weighted, two with equal weights, one irrelevant file, is uninstall",
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
			kotsCharts: []v1beta1.HelmChart{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "chart1",
					},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "chart1",
							ChartVersion: "ver1",
						},
						Weight: 1,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "chart2",
					},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "chart2",
							ChartVersion: "v1",
						},
						Weight: 1,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "chart3",
					},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "chart3",
							ChartVersion: "v1",
						},
						Weight: 5,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "chart4",
					},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "chart4",
							ChartVersion: "v1",
						},
					},
				},
			},
			isUninstall: true,
			want: []orderedDir{
				{
					Name:         "chart3",
					Weight:       5,
					ChartName:    "chart3",
					ChartVersion: "v1",
					ReleaseName:  "chart3",
				},
				{
					Name:         "chart2",
					Weight:       1,
					ChartName:    "chart2",
					ChartVersion: "v1",
					ReleaseName:  "chart2",
				},
				{
					Name:         "chart1",
					Weight:       1,
					ChartName:    "chart1",
					ChartVersion: "ver1",
					ReleaseName:  "chart1",
				},
				{
					Name:         "chart4",
					ChartName:    "chart4",
					ChartVersion: "v1",
					ReleaseName:  "chart4",
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
			kotsCharts: []v1beta1.HelmChart{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "chart1",
					},
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
					Name:         "chart1",
					Weight:       -5,
					ChartName:    "c1",
					ChartVersion: "ver1",
					ReleaseName:  "c1",
				},
				{
					Name:        "chart2",
					ChartName:   "c2",
					ReleaseName: "c2",
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
			kotsCharts: []v1beta1.HelmChart{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "chart1",
					},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "generic",
							ChartVersion: "ver1",
						},
						Weight: -1,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "chart2",
					},
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
					Name:         "chart1",
					ChartName:    "generic",
					ChartVersion: "ver1",
					ReleaseName:  "generic",
					Weight:       -1,
				},
				{
					Name:         "chart2",
					Weight:       2,
					ChartName:    "generic",
					ChartVersion: "ver2",
					ReleaseName:  "generic",
				},
			},
		},
		{
			name: "metadata name does not match directory name",
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
			kotsCharts: []v1beta1.HelmChart{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "chart3",
					},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "generic",
							ChartVersion: "ver1",
						},
						Weight: -1,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "chart4",
					},
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
					Name:         "chart1",
					ChartName:    "generic",
					ChartVersion: "ver1",
					ReleaseName:  "generic",
				},
				{
					Name:         "chart2",
					ChartName:    "generic",
					ChartVersion: "ver2",
					ReleaseName:  "generic",
				},
			},
		},
		{
			name: "kots chart specifies a release name",
			files: []file{
				{
					path: "rel1/Chart.yaml",
					contents: `
name: generic
version: ver1
`,
				},
				{
					path: "rel2/Chart.yaml",
					contents: `
name: generic
version: ver2
`,
				},
			},
			kotsCharts: []v1beta1.HelmChart{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "chart1",
					},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "generic",
							ChartVersion: "ver1",
							ReleaseName:  "rel1",
						},
						Weight: -1,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "chart2",
					},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "generic",
							ChartVersion: "ver2",
							ReleaseName:  "rel2",
						},
						Weight: 2,
					},
				},
			},
			want: []orderedDir{
				{
					Name:         "rel1",
					ChartName:    "generic",
					ChartVersion: "ver1",
					ReleaseName:  "rel1",
					Weight:       -1,
				},
				{
					Name:         "rel2",
					ChartName:    "generic",
					ChartVersion: "ver2",
					ReleaseName:  "rel2",
					Weight:       2,
				},
			},
		},
		{
			name: "kots chart specifies helm flags",
			files: []file{
				{
					path: "chart1/Chart.yaml",
					contents: `
name: generic1
version: ver1
`,
				},
				{
					path: "chart2/Chart.yaml",
					contents: `
name: generic2
version: ver2
`,
				},
			},
			kotsCharts: []v1beta1.HelmChart{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "chart1",
					},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "generic1",
							ChartVersion: "ver1",
						},
						HelmUpgradeFlags: []string{
							"--skip-crds",
							"--no-hooks",
							"--atomic",
							"--description=my description",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "chart2",
					},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "generic2",
							ChartVersion: "ver2",
						},
					},
				},
			},
			want: []orderedDir{
				{
					Name:         "chart1",
					ChartName:    "generic1",
					ChartVersion: "ver1",
					ReleaseName:  "generic1",
					UpgradeFlags: []string{
						"--skip-crds",
						"--no-hooks",
						"--atomic",
						"--description=my description",
					},
				},
				{
					Name:         "chart2",
					ChartName:    "generic2",
					ChartVersion: "ver2",
					ReleaseName:  "generic2",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			tempdir := t.TempDir()

			// populate host directory
			for _, file := range tt.files {
				err := os.MkdirAll(filepath.Dir(filepath.Join(tempdir, file.path)), os.ModePerm)
				req.NoError(err)

				err = ioutil.WriteFile(filepath.Join(tempdir, file.path), []byte(file.contents), os.ModePerm)
				req.NoError(err)
			}

			got, err := getSortedCharts(tempdir, tt.kotsCharts, tt.targetNamespace, tt.isUninstall)
			req.NoError(err)
			req.Equal(tt.want, got)
		})
	}
}
