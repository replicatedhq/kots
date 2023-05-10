package client

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta2"
	"github.com/replicatedhq/kots/pkg/kotsutil"
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
		v1Beta1Files    []file
		v1Beta2Files    []file
		kotsCharts      []kotsutil.HelmChartInterface
		targetNamespace string
		isUninstall     bool
		want            []orderedDir
	}{
		{
			name: "chart without an entry in kotsCharts should work", // this should not come up in practice but is good to reduce risk
			v1Beta1Files: []file{
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
					APIVersion:  "kots.io/v1beta1",
				},
			},
		},
		{
			name: "four charts, one not weighted, two with equal weights, one irrelevant file",
			v1Beta1Files: []file{
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
			kotsCharts: []kotsutil.HelmChartInterface{
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
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
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
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
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
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
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
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
					APIVersion:   "kots.io/v1beta1",
				},
				{
					Name:         "chart1",
					Weight:       1,
					ChartName:    "chart1",
					ChartVersion: "ver1",
					ReleaseName:  "chart1",
					APIVersion:   "kots.io/v1beta1",
				},
				{
					Name:         "chart2",
					Weight:       1,
					ChartName:    "chart2",
					ChartVersion: "v1",
					ReleaseName:  "chart2",
					APIVersion:   "kots.io/v1beta1",
				},
				{
					Name:         "chart3",
					Weight:       5,
					ChartName:    "chart3",
					ChartVersion: "v1",
					ReleaseName:  "chart3",
					APIVersion:   "kots.io/v1beta1",
				},
			},
		},
		{
			name: "four charts, one not weighted, two with equal weights, one irrelevant file, is uninstall",
			v1Beta1Files: []file{
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
			kotsCharts: []kotsutil.HelmChartInterface{
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
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
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
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
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
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
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
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
					APIVersion:   "kots.io/v1beta1",
				},
				{
					Name:         "chart2",
					Weight:       1,
					ChartName:    "chart2",
					ChartVersion: "v1",
					ReleaseName:  "chart2",
					APIVersion:   "kots.io/v1beta1",
				},
				{
					Name:         "chart1",
					Weight:       1,
					ChartName:    "chart1",
					ChartVersion: "ver1",
					ReleaseName:  "chart1",
					APIVersion:   "kots.io/v1beta1",
				},
				{
					Name:         "chart4",
					ChartName:    "chart4",
					ChartVersion: "v1",
					ReleaseName:  "chart4",
					APIVersion:   "kots.io/v1beta1",
				},
			},
		},
		{
			name: "negative weights before no weight",
			v1Beta1Files: []file{
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
			kotsCharts: []kotsutil.HelmChartInterface{
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
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
					APIVersion:   "kots.io/v1beta1",
				},
				{
					Name:        "chart2",
					ChartName:   "c2",
					ReleaseName: "c2",
					APIVersion:  "kots.io/v1beta1",
				},
			},
		},
		{
			name: "same name, different versions",
			v1Beta1Files: []file{
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
			kotsCharts: []kotsutil.HelmChartInterface{
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
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
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
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
					APIVersion:   "kots.io/v1beta1",
				},
				{
					Name:         "chart2",
					Weight:       2,
					ChartName:    "generic",
					ChartVersion: "ver2",
					ReleaseName:  "generic",
					APIVersion:   "kots.io/v1beta1",
				},
			},
		},
		{
			name: "metadata name does not match directory name",
			v1Beta1Files: []file{
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
			kotsCharts: []kotsutil.HelmChartInterface{
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
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
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
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
					APIVersion:   "kots.io/v1beta1",
				},
				{
					Name:         "chart2",
					ChartName:    "generic",
					ChartVersion: "ver2",
					ReleaseName:  "generic",
					APIVersion:   "kots.io/v1beta1",
				},
			},
		},
		{
			name: "kots chart specifies a release name",
			v1Beta1Files: []file{
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
			kotsCharts: []kotsutil.HelmChartInterface{
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
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
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
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
					APIVersion:   "kots.io/v1beta1",
				},
				{
					Name:         "rel2",
					ChartName:    "generic",
					ChartVersion: "ver2",
					ReleaseName:  "rel2",
					Weight:       2,
					APIVersion:   "kots.io/v1beta1",
				},
			},
		},
		{
			name: "kots chart specifies helm flags",
			v1Beta1Files: []file{
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
			kotsCharts: []kotsutil.HelmChartInterface{
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
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
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
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
					APIVersion: "kots.io/v1beta1",
				},
				{
					Name:         "chart2",
					ChartName:    "generic2",
					ChartVersion: "ver2",
					ReleaseName:  "generic2",
					APIVersion:   "kots.io/v1beta1",
				},
			},
		},
		{
			name: "v1beta2 chart",
			v1Beta2Files: []file{
				{
					path:     "minimal-release/minimal-0.0.1.tgz",
					contents: "H4sIFAAAAAAA/ykAK2FIUjBjSE02THk5NWIzVjBkUzVpWlM5Nk9WVjZNV2xqYW5keVRRbz1IZWxtAOzSsQoCMQwG4M59ij5B/dvECrf6Du4ZDixcq/TOA99eEF3O0YII+ZZ/yJA/kJJrLjLtjmdpi79LmUx3AJCYnwlgm8CeTWCO6UBEiQxCTBSMQ/8qn27zIs3g613b4/6EXPNpbHO+1MGt0VYp4+BeT2HX9wQePthfd1VKKdXPIwAA//8d5AfYAAgAAA==",
				},
			},
			kotsCharts: []kotsutil.HelmChartInterface{
				&v1beta2.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta2",
						Kind:       "HelmChart",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "minimal",
					},
					Spec: v1beta2.HelmChartSpec{
						Chart: v1beta2.ChartIdentifier{
							Name:         "minimal",
							ChartVersion: "0.0.1",
						},
						ReleaseName: "minimal-release",
						Namespace:   "my-namespace",
						HelmUpgradeFlags: []string{
							"--skip-crds",
						},
					},
				},
			},
			want: []orderedDir{
				{
					Name:         "minimal-release",
					ChartName:    "minimal",
					ChartVersion: "0.0.1",
					ReleaseName:  "minimal-release",
					Namespace:    "my-namespace",
					UpgradeFlags: []string{
						"--skip-crds",
					},
					APIVersion: "kots.io/v1beta2",
				},
			},
		},
		{
			name: "v1beta2 charts with weights",
			v1Beta2Files: []file{
				{
					path:     "minimal-release-1/minimal-0.0.1.tgz",
					contents: "H4sIFAAAAAAA/ykAK2FIUjBjSE02THk5NWIzVjBkUzVpWlM5Nk9WVjZNV2xqYW5keVRRbz1IZWxtAOzSsQoCMQwG4M59ij5B/dvECrf6Du4ZDixcq/TOA99eEF3O0YII+ZZ/yJA/kJJrLjLtjmdpi79LmUx3AJCYnwlgm8CeTWCO6UBEiQxCTBSMQ/8qn27zIs3g613b4/6EXPNpbHO+1MGt0VYp4+BeT2HX9wQePthfd1VKKdXPIwAA//8d5AfYAAgAAA==",
				},
				{
					path:     "minimal-release-2/minimal-0.0.1.tgz",
					contents: "H4sIFAAAAAAA/ykAK2FIUjBjSE02THk5NWIzVjBkUzVpWlM5Nk9WVjZNV2xqYW5keVRRbz1IZWxtAOzSsQoCMQwG4M59ij5B/dvECrf6Du4ZDixcq/TOA99eEF3O0YII+ZZ/yJA/kJJrLjLtjmdpi79LmUx3AJCYnwlgm8CeTWCO6UBEiQxCTBSMQ/8qn27zIs3g613b4/6EXPNpbHO+1MGt0VYp4+BeT2HX9wQePthfd1VKKdXPIwAA//8d5AfYAAgAAA==",
				},
			},
			kotsCharts: []kotsutil.HelmChartInterface{
				&v1beta2.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta2",
						Kind:       "HelmChart",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "minimal-1",
					},
					Spec: v1beta2.HelmChartSpec{
						Chart: v1beta2.ChartIdentifier{
							Name:         "minimal",
							ChartVersion: "0.0.1",
						},
						ReleaseName: "minimal-release-1",
						Weight:      2,
					},
				},
				&v1beta2.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta2",
						Kind:       "HelmChart",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "minimal-2",
					},
					Spec: v1beta2.HelmChartSpec{
						Chart: v1beta2.ChartIdentifier{
							Name:         "minimal",
							ChartVersion: "0.0.1",
						},
						ReleaseName: "minimal-release-2",
						Weight:      1,
					},
				},
			},
			want: []orderedDir{
				{
					Name:         "minimal-release-2",
					ChartName:    "minimal",
					ChartVersion: "0.0.1",
					Weight:       1,
					ReleaseName:  "minimal-release-2",
					APIVersion:   "kots.io/v1beta2",
				},
				{
					Name:         "minimal-release-1",
					ChartName:    "minimal",
					ChartVersion: "0.0.1",
					Weight:       2,
					ReleaseName:  "minimal-release-1",
					APIVersion:   "kots.io/v1beta2",
				},
			},
		},
		{
			name: "v1beat1 and v1beta2 charts with weights",
			v1Beta1Files: []file{
				{
					path: "generic1-release/Chart.yaml",
					contents: `
name: generic1
version: ver1
`,
				},
				{
					path: "generic2-release/Chart.yaml",
					contents: `
name: generic2
version: ver2
`,
				},
			},
			v1Beta2Files: []file{
				{
					path:     "minimal-release-1/minimal-0.0.1.tgz",
					contents: "H4sIFAAAAAAA/ykAK2FIUjBjSE02THk5NWIzVjBkUzVpWlM5Nk9WVjZNV2xqYW5keVRRbz1IZWxtAOzSsQoCMQwG4M59ij5B/dvECrf6Du4ZDixcq/TOA99eEF3O0YII+ZZ/yJA/kJJrLjLtjmdpi79LmUx3AJCYnwlgm8CeTWCO6UBEiQxCTBSMQ/8qn27zIs3g613b4/6EXPNpbHO+1MGt0VYp4+BeT2HX9wQePthfd1VKKdXPIwAA//8d5AfYAAgAAA==",
				},
				{
					path:     "minimal-release-2/minimal-0.0.1.tgz",
					contents: "H4sIFAAAAAAA/ykAK2FIUjBjSE02THk5NWIzVjBkUzVpWlM5Nk9WVjZNV2xqYW5keVRRbz1IZWxtAOzSsQoCMQwG4M59ij5B/dvECrf6Du4ZDixcq/TOA99eEF3O0YII+ZZ/yJA/kJJrLjLtjmdpi79LmUx3AJCYnwlgm8CeTWCO6UBEiQxCTBSMQ/8qn27zIs3g613b4/6EXPNpbHO+1MGt0VYp4+BeT2HX9wQePthfd1VKKdXPIwAA//8d5AfYAAgAAA==",
				},
			},
			kotsCharts: []kotsutil.HelmChartInterface{
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "generic1",
					},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "generic1",
							ChartVersion: "ver1",
							ReleaseName:  "generic1-release",
						},
						Weight: 2,
					},
				},
				&v1beta1.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChart",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "generic2",
					},
					Spec: v1beta1.HelmChartSpec{
						Chart: v1beta1.ChartIdentifier{
							Name:         "generic2",
							ChartVersion: "ver2",
							ReleaseName:  "generic2-release",
						},
						Weight: 1,
					},
				},
				&v1beta2.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta2",
						Kind:       "HelmChart",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "minimal-1",
					},
					Spec: v1beta2.HelmChartSpec{
						Chart: v1beta2.ChartIdentifier{
							Name:         "minimal",
							ChartVersion: "0.0.1",
						},
						ReleaseName: "minimal-release-1",
						Weight:      2,
					},
				},
				&v1beta2.HelmChart{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta2",
						Kind:       "HelmChart",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "minimal-2",
					},
					Spec: v1beta2.HelmChartSpec{
						Chart: v1beta2.ChartIdentifier{
							Name:         "minimal",
							ChartVersion: "0.0.1",
						},
						ReleaseName: "minimal-release-2",
						Weight:      1,
					},
				},
			},
			want: []orderedDir{
				{
					Name:         "generic2-release",
					ChartName:    "generic2",
					ChartVersion: "ver2",
					Weight:       1,
					ReleaseName:  "generic2-release",
					APIVersion:   "kots.io/v1beta1",
				},
				{
					Name:         "minimal-release-2",
					ChartName:    "minimal",
					ChartVersion: "0.0.1",
					Weight:       1,
					ReleaseName:  "minimal-release-2",
					APIVersion:   "kots.io/v1beta2",
				},
				{
					Name:         "generic1-release",
					ChartName:    "generic1",
					ChartVersion: "ver1",
					Weight:       2,
					ReleaseName:  "generic1-release",
					APIVersion:   "kots.io/v1beta1",
				},
				{
					Name:         "minimal-release-1",
					ChartName:    "minimal",
					ChartVersion: "0.0.1",
					Weight:       2,
					ReleaseName:  "minimal-release-1",
					APIVersion:   "kots.io/v1beta2",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			v1Beta1ChartsDir := t.TempDir()
			v1Beta2ChartsDir := t.TempDir()

			// populate host directory
			for _, file := range tt.v1Beta1Files {
				err := os.MkdirAll(filepath.Dir(filepath.Join(v1Beta1ChartsDir, file.path)), os.ModePerm)
				req.NoError(err)

				err = ioutil.WriteFile(filepath.Join(v1Beta1ChartsDir, file.path), []byte(file.contents), os.ModePerm)
				req.NoError(err)
			}

			for _, file := range tt.v1Beta2Files {
				err := os.MkdirAll(filepath.Dir(filepath.Join(v1Beta2ChartsDir, file.path)), os.ModePerm)
				req.NoError(err)

				decoded, err := base64.StdEncoding.DecodeString(file.contents)
				req.NoError(err)

				err = ioutil.WriteFile(filepath.Join(v1Beta2ChartsDir, file.path), decoded, os.ModePerm)
				req.NoError(err)
			}

			got, err := getSortedCharts(v1Beta1ChartsDir, v1Beta2ChartsDir, tt.kotsCharts, tt.targetNamespace, tt.isUninstall)
			req.NoError(err)
			req.Equal(tt.want, got)
		})
	}
}
