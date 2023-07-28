package base

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
)

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
					Name: "test-chart",
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
						Content: []byte("apiVersion: v1\ndata:\n  release: SDRzSUFBQUFBQUFDLzdTU1gydmJNQlRGdjRxNGU1VVQyeU9EQ3Zvd3Qyc29iQWx0V2V4NEhrT1JsVVNML21ISlRrM3dkeDl1NXFTdzVXR3d2dWtlWGM3OWNlNDlnS2FLQXdIUG5RL1lsbFllTUFpOU5rQU9zQmFWOHo5S2JxVnBlUWtFQUlPa2YwZ2xsOXlmQzhjcVliMHdHZ2pjVmkycWFvMllVYlp2QWd6T1UxODdJSEF5NlRBY0o1TURLTzVwU1QzdDMzOURhM2psanRiaEtCcUZnSUZhc1RpSlRkUzdTY04yUUhRdEpRYlBsWlhVY3dmazJ5dkwzK0w0Q0JGRW81WXEyZU8veklabGx0aUZXclFzbHMzcXA5a3NzMlRQMnF2NHk0T3RhRHJaelVYeUtjOFN0M292Zlo1T3doc1Z5WEo2dDF0bWo5djV4bXp1cDVQdEt2MzY0WDc2S05uMHVlRlA0Zk9OK0xoWnFUdWZaN1A5TXAzSnVVakNQSnVGbjlPck9uKzR2b1lPWHdhTTN4NndoZTQ3aG9iS3VnL3JtSjVqVzY3b1VLMkZITDQ2RElwcXNlYk9BNEVnQ0FyOURqMlp1bUtjb1BQR3hoZVNMdlI1YlFSUmE5MjRpUXE5RTdvazZQYWxVM0h0Q3ozY0F5azBRbjA0QkExR2crSXNQUTAxbWhmNkgyamkvMFVURi9yMWRVWVlUbWpEQ2Z1OWdlNVhBQUFBLy8rVzFhd1RjUU1BQUE9PQ==\nkind: Secret\nmetadata:\n  creationTimestamp: null\n  labels:\n    createdAt: \"1\"\n    name: test-chart\n    owner: helm\n    status: deployed\n    version: \"1\"\n  name: sh.helm.release.v1.test-chart.v1\n  namespace: test-two\ntype: helm.sh/release.v1\n"),
					},
					{
						Path: "deploy-1.yaml",
						Content: []byte("# Source: test-chart/templates/deploy-1.yaml\napiVersion: apps/v1\nkind: Deployment\n" +
							"metadata:\n  name: deploy-1\n  namespace: test-one\n"),
					},
					{
						Path:    "deploy-2.yaml",
						Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2\n  namespace: test-two\n"),
					},
				},
				AdditionalFiles: []BaseFile{
					{
						Path:    "values.yaml",
						Content: []byte("{}\n"),
					},
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
					Name: "test-chart",
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
						Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2\n  namespace: test-two\n"),
					},
				},
				AdditionalFiles: []BaseFile{
					{
						Path:    "values.yaml",
						Content: []byte("{}\n"),
					},
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
						Content: []byte("apiVersion: v1\ndata:\n  release: SDRzSUFBQUFBQUFDLzd4VFhXdmJNQlQ5SytidVZVN3NsQXdxNk1QY3JxR3NTMmpDWXNmekdMS3NKRnIwaFNYYk5TWC9mVGlPbThLNngrMUp1a2VYY3c3M0hyMkFJcElCUGgzV0VNcDh4NndEQkZ4dE5lQVgyUExTdXA4Rk0wSzNyQUFNZ0VDUVA2Q0NDZVl1aGFVbE40NXJCUmp1eXRZcksrVlJMVTNYQkFpc0k2NnlnT0dWNUlpQTdrbnBPa25KSENtSUk5MzliSzh6NWZjTkNHcFcycDQ2R0lXakFCQVF3OWV2WUIxMmJFTFRBMkJWQ1lIQU1Xa0VjY3dDL3Y2RzhneU9leE9qbGtqUm1UOHB3eWFKekZxdVd6b1JkZjVMN3paSjFORDJldkwxeVpRa25oNFdQUHFjSnBITnI0Ukw0Mmx3SzBOUnpPNFBtMlM1WCt6MDdtRTIzZWZ4dDQ4UHM2V2dzK2VhcllMblcvNXBsOHQ3bHlielpoUFB4WUpIUVpyTWc4ZjR1a3FmdEh0Y0JWLytnVzc3Vjkxa2VaVTNOemR3L0lHZ0pxTHFKdFNQek5JOWsyU290bHdNVDBjRWtpaSs3VktDd2ZmOVRIM3dWcm9xS2NQZVpVM2pkOGVicWN1bXNFZU1zZU02ek5TQnF3SjdkNmMreVpUTDFCQUJuQ25QNnhhR3ZaN0dEd2ZrbE5henBGWXNVLy9meStROUw2N1JtWHFiMGhCZGZ0Y1FaZGRvT1A0T0FBRC8vd0RtQjhkOUF3QUE=\nkind: Secret\nmetadata:\n  creationTimestamp: null\n  labels:\n    createdAt: \"1\"\n    name: namespace-test\n    owner: helm\n    status: deployed\n    version: \"1\"\n  name: sh.helm.release.v1.namespace-test.v1\n  namespace: test-two\ntype: helm.sh/release.v1\n"),
					},
					{
						Path: "deploy.yaml",
						Content: []byte("# Source: test-chart/templates/deploy.yaml\n" +
							"apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-1\n  namespace: test-one\n" +
							"\n---\n" +
							"# Source: test-chart/templates/deploy.yaml\n" +
							"apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2\n  namespace: test-two\n"),
					},
				},
				AdditionalFiles: []BaseFile{
					{
						Path:    "values.yaml",
						Content: []byte("{}\n"),
					},
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
							"\n---\n" +
							"apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2\n  namespace: test-two\n"),
					},
				},
				AdditionalFiles: []BaseFile{
					{
						Path:    "values.yaml",
						Content: []byte("{}\n"),
					},
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
						Content: []byte("invalid\n\nyaml\n"),
					},
				},
				AdditionalFiles: []BaseFile{
					{
						Path:    "values.yaml",
						Content: []byte("{}\n"),
					},
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
						Content: []byte("apiVersion: v1\ndata:\n  release: SDRzSUFBQUFBQUFDLzJTU1cyL2FNQlRIdjRwMTl1cTBCRGFrV3RyREJpeGkwNWpXaVhCWnBzbllKcmo0SnRzSlN5dSsrMlFvcFZLZmtuUFIvL3pQK2ZrSkROVUN5T2tUSEdVaWl5SkV3Q0ROMWdKNWdxMzBJZjdsd2luYkNRNEVBSU9pYjFKY0tCR3ZRV0JldWlpdEFRSmozeUhmR01Tc2Rxa0pNSVJJWXhPQXdJdklFUVBiVVIvVFNDMGk1VFRTOVA5c0w1bkt6ZzBZV3VIRFdicDNrOS8wQUFOMXNueEp0bmxTVTVidGdaaEdLUXhSYUtkb0ZBSEk3MWVTejhsYjV2bE5SN1ZLems5alliWDg3RXBkZHF5djJzMkRyZm5ENUJ2dHEyWTl0dlhQUWZuSWk3czQxK1hqWmxCMnEzNDVXUy9XYnFOVmp5N3VtcEhPRlMrKzdGZkwrOTJQMnRiVDRzTnVzNWdQcDBYNWZyWElENXRpSGxlRHIvdVJtUjNXaSsvRGtmeFVzLzZzWmNWOE9KM01BbC9PZXV2bDlDTWMvMkJvcVdxUzdmTWVnZTJFcHBkb0s5V2xkTVNncVpIYmhJNUFsbVdWZVlkKzJjWXpRZEQxZHJkdmQ2N005WFlFdFhsbDl0Sndna1pOaUZiZmkzQVNHWXV0TkRMeHJNeUZEcWtNUXVtV0JJbC9OS0hObU9lVkNVNndVeTB3NndSQkk5V0VLSHhsWHBQTDhmWEZYZkRHZzRYai93QUFBUC8vSVVPakpaRUNBQUE9\nkind: Secret\nmetadata:\n  creationTimestamp: null\n  labels:\n    createdAt: \"1\"\n    name: namespace-test\n    owner: helm\n    status: deployed\n    version: \"1\"\n  name: sh.helm.release.v1.namespace-test.v1\n  namespace: test-two\ntype: helm.sh/release.v1\n"),
					},
					{
						Path:    "crd.yaml",
						Content: []byte("# Source: test-chart/templates/crd.yaml\napiVersion: v1\nkind: CustomResourceDefinition\nmetadata:\n  name: example-crd\nspec:\n  scope: Cluster\n"),
					},
				},
				AdditionalFiles: []BaseFile{
					{
						Path:    "values.yaml",
						Content: []byte("{}\n"),
					},
					{
						Path:    "Chart.yaml",
						Content: []byte("name: test-chart\nversion: 0.1.0"),
					},
				},
			},
		},
		{
			name: "test subcharts with namespace",
			args: args{
				upstream: &upstreamtypes.Upstream{
					Name: "test-chart",
					Files: []upstreamtypes.UpstreamFile{
						{
							Path:    "Chart.yaml",
							Content: []byte("name: test-chart\nversion: 0.1.0"),
						},
						{
							Path:    "templates/deploy-1.yaml",
							Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-1"),
						},
						{
							Path:    "charts/test-subchart/Chart.yaml",
							Content: []byte("name: test-subchart\nversion: 0.2.0"),
						},
						{
							Path:    "charts/test-subchart/templates/deploy-2.yaml",
							Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2"),
						},
					},
				},
				renderOptions: &RenderOptions{
					HelmVersion:    "v3",
					Namespace:      "test-namespace",
					UseHelmInstall: true,
				},
			},
			want: &Base{
				Namespace: "test-namespace",
				Files: []BaseFile{
					{
						Path:    "templates/deploy-1.yaml",
						Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-1\n  namespace: test-namespace\n"),
					},
				},
				AdditionalFiles: []BaseFile{
					{
						Path:    "values.yaml",
						Content: []byte("test-subchart:\n  global: {}\n"),
					},
					{
						Path:    "Chart.yaml",
						Content: []byte("name: test-chart\nversion: 0.1.0"),
					},
				},
				Bases: []Base{
					{
						Namespace: "test-namespace",
						Path:      "charts/test-subchart",
						Files: []BaseFile{
							{
								Path:    "templates/deploy-2.yaml",
								Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2\n  namespace: test-namespace\n"),
							},
						},
						AdditionalFiles: []BaseFile{
							{
								Path:    "Chart.yaml",
								Content: []byte("name: test-subchart\nversion: 0.2.0"),
							},
						},
					},
				},
			},
		},
		{
			name: "test subcharts with no templates dir and two charts",
			args: args{
				upstream: &upstreamtypes.Upstream{
					Name: "test-chart",
					Files: []upstreamtypes.UpstreamFile{
						{
							Path:    "Chart.yaml",
							Content: []byte("name: test-chart\nversion: 0.1.0"),
						},

						{
							Path:    "charts/test-subchart/Chart.yaml",
							Content: []byte("name: test-subchart\nversion: 0.2.0"),
						},
						{
							Path:    "charts/test-subchart/templates/deploy-2.yaml",
							Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2"),
						},
						{
							Path:    "charts/test-subchart-2/Chart.yaml",
							Content: []byte("name: test-subchart-2\nversion: 0.2.0"),
						},
						{
							Path:    "charts/test-subchart-2/templates/deploy-3.yaml",
							Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-3"),
						},
					},
				},
				renderOptions: &RenderOptions{
					HelmVersion:    "v3",
					Namespace:      "test-namespace",
					UseHelmInstall: true,
				},
			},
			want: &Base{
				Namespace: "test-namespace",

				AdditionalFiles: []BaseFile{
					{
						Path:    "values.yaml",
						Content: []byte("test-subchart:\n  global: {}\ntest-subchart-2:\n  global: {}\n"),
					},
					{
						Path:    "Chart.yaml",
						Content: []byte("name: test-chart\nversion: 0.1.0"),
					},
				},
				Bases: []Base{
					{
						Namespace: "test-namespace",
						Path:      "charts/test-subchart-2",
						Files: []BaseFile{
							{
								Path:    "templates/deploy-3.yaml",
								Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-3\n  namespace: test-namespace\n"),
							},
						},
						AdditionalFiles: []BaseFile{
							{
								Path:    "Chart.yaml",
								Content: []byte("name: test-subchart-2\nversion: 0.2.0"),
							},
						},
					},
					{
						Namespace: "test-namespace",
						Path:      "charts/test-subchart",
						Files: []BaseFile{
							{
								Path:    "templates/deploy-2.yaml",
								Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2\n  namespace: test-namespace\n"),
							},
						},
						AdditionalFiles: []BaseFile{
							{
								Path:    "Chart.yaml",
								Content: []byte("name: test-subchart\nversion: 0.2.0"),
							},
						},
					},
				},
			},
		},
		{
			name: "test subcharts with templates dir and two charts",
			args: args{
				upstream: &upstreamtypes.Upstream{
					Name: "test-chart",
					Files: []upstreamtypes.UpstreamFile{
						{
							Path:    "Chart.yaml",
							Content: []byte("name: test-chart\nversion: 0.1.0"),
						},
						{
							Path:    "templates/deploy-2.yaml",
							Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2"),
						},

						{
							Path:    "charts/test-subchart/Chart.yaml",
							Content: []byte("name: test-subchart\nversion: 0.2.0"),
						},
						{
							Path:    "charts/test-subchart/templates/deploy-2.yaml",
							Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2"),
						},
						{
							Path:    "charts/test-subchart-2/Chart.yaml",
							Content: []byte("name: test-subchart-2\nversion: 0.2.0"),
						},
						{
							Path:    "charts/test-subchart-2/templates/deploy-3.yaml",
							Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-3"),
						},
					},
				},
				renderOptions: &RenderOptions{
					HelmVersion:    "v3",
					Namespace:      "",
					UseHelmInstall: true,
				},
			},
			want: &Base{
				Namespace: "",
				Files: []BaseFile{
					{
						Path:    "templates/deploy-2.yaml",
						Content: []byte("# Source: test-chart/templates/deploy-2.yaml\napiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2\n"),
					},
				},
				AdditionalFiles: []BaseFile{
					{
						Path:    "values.yaml",
						Content: []byte("test-subchart:\n  global: {}\ntest-subchart-2:\n  global: {}\n"),
					},
					{
						Path:    "Chart.yaml",
						Content: []byte("name: test-chart\nversion: 0.1.0"),
					},
				},
				Bases: []Base{
					{
						Namespace: "",
						Path:      "charts/test-subchart-2",
						Files: []BaseFile{
							{
								Path:    "templates/deploy-3.yaml",
								Content: []byte("# Source: test-chart/charts/test-subchart-2/templates/deploy-3.yaml\napiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-3\n"),
							},
						},
						AdditionalFiles: []BaseFile{
							{
								Path:    "Chart.yaml",
								Content: []byte("name: test-subchart-2\nversion: 0.2.0"),
							},
						},
					},
					{
						Namespace: "",
						Path:      "charts/test-subchart",
						Files: []BaseFile{
							{
								Path:    "templates/deploy-2.yaml",
								Content: []byte("# Source: test-chart/charts/test-subchart/templates/deploy-2.yaml\napiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2\n"),
							},
						},
						AdditionalFiles: []BaseFile{
							{
								Path:    "Chart.yaml",
								Content: []byte("name: test-subchart\nversion: 0.2.0"),
							},
						},
					},
				},
			},
		},
		{
			name: "helm v3 configmap no newline",
			args: args{
				upstream: &upstreamtypes.Upstream{
					Name: "test-chart",
					Files: []upstreamtypes.UpstreamFile{
						{
							Path:    "Chart.yaml",
							Content: []byte("name: test-chart\nversion: 0.1.0"),
						},
						{
							Path:    "templates/configmap.yaml",
							Content: []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: config-map\ndata:\n  key: |\n    multiline\n    configmap\n    data"),
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
						Content: []byte("apiVersion: v1\ndata:\n  release: SDRzSUFBQUFBQUFDLzNTU1gyK2JNQlRGdjRwMTl3cHBpTFJLODF0Q0Y0cWliV3JYcFlFeFRRWU1jZkUvWVVPRXNuejN5YUVobGJhKytaeHJmdGYzWEk0Z2lhQ0F3VkpqL1dKUFdnc2VNRmtwd0Vlb1dHdnM3NUpxcmdaYUFnYndnSk4vckpKeWFxL0NGQzNUbGlrSkdPN2FBYldkUklVUzJsMENENHdsdGpPQVlZS2NQQmc3NHlNSWFrbEpMSEhuL3oydHA2MFowZk5aTUp1REIwU3o3V1QyZ2FOeFZUU0FaY2U1QjVZS3pZbWxCdkRQTjhoWDg2WlFzbUsxSUhvMkVNSGQrOC9OSWRtdDlGWnNoMkxCKy94RjFlWEw1dzFaOEM2OVUvWEQ0bE9YQ2k2Zm50ZUhVQVM4ak5aTnNudmNmNnRWSFVjZjkvbnpqOXM0K3Rybkl0WHBFTmhrdDl5azBYcWVQS2xOSEs3YWROZmN4dmVIVFJ3dTZ6Z0tndnorVWVjUjc5SUhWVHN2ZWVYbmpzK1dkUnl1em53NC9mS2dKN3h6dzR6VG1XSlBCYm1vaXZGTDZlU0JJSkpWMUZqQTRQdCtKaitnNzZwckM0clJOZEdiOTVMSTVEVlhqUG9na3cyVEpVYmgrZFlYb2pONVdSWE9KRUl1V0l4R2hpOWNlU28xZE1Eb2p6c2hKRHB1R1dlU2puTHFPVXIzU1NiZmJqbnd6aXN6bWhUVHIyQVBDazUvQXdBQS8vOTBDc1BCdVFJQUFBPT0=\nkind: Secret\nmetadata:\n  creationTimestamp: null\n  labels:\n    createdAt: \"1\"\n    name: test-chart\n    owner: helm\n    status: deployed\n    version: \"1\"\n  name: sh.helm.release.v1.test-chart.v1\n  namespace: test-two\ntype: helm.sh/release.v1\n"),
					},
					{
						Path:    "configmap.yaml",
						Content: []byte("apiVersion: v1\ndata:\n  key: |\n    multiline\n    configmap\n    data\nkind: ConfigMap\nmetadata:\n  name: config-map\n  namespace: test-two\n"),
					},
				},
				AdditionalFiles: []BaseFile{
					{
						Path:    "values.yaml",
						Content: []byte("{}\n"),
					},
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

func Test_pathToCharts(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "top-level path",
			path: "",
			want: []string{""},
		},
		{
			name: "subchart path",
			path: "charts/subchart",
			want: []string{"", "subchart"},
		},
		{
			name: "subsubchart path",
			path: "charts/subchart/charts/subsubchart",
			want: []string{"", "subchart", "subsubchart"},
		},
		{
			name: "subchart that has 'charts' in the name as a suffix",
			path: "charts/subchart-charts/charts/subsubchart",
			want: []string{"", "subchart-charts", "subsubchart"},
		},
		{
			name: "subsubchart that has 'charts' in the name as a suffix",
			path: "charts/subchart/charts/subsubchart-charts",
			want: []string{"", "subchart", "subsubchart-charts"},
		},
		{
			name: "subchart and subsubchart that have 'charts' in the name as a suffix",
			path: "charts/subchart-charts/charts/subsubchart-charts",
			want: []string{"", "subchart-charts", "subsubchart-charts"},
		},
		{
			name: "subchart that has 'charts' in the name as a prefix",
			path: "charts/charts-subchart/charts/subsubchart",
			want: []string{"", "charts-subchart", "subsubchart"},
		},
		{
			name: "subsubchart that has 'charts' in the name as a prefix",
			path: "charts/subchart/charts/charts-subsubchart",
			want: []string{"", "subchart", "charts-subsubchart"},
		},
		{
			name: "subchart and subsubchart that have 'charts' in the name as a prefix",
			path: "charts/charts-subchart/charts/charts-subsubchart",
			want: []string{"", "charts-subchart", "charts-subsubchart"},
		},
		{
			name: "subchart that has 'charts' in the name",
			path: "charts/sub-charts-chart/charts/subsubchart",
			want: []string{"", "sub-charts-chart", "subsubchart"},
		},
		{
			name: "subsubchart that has 'charts' in the name",
			path: "charts/subchart/charts/subsub-charts-chart",
			want: []string{"", "subchart", "subsub-charts-chart"},
		},
		{
			name: "subchart and subsubchart that have 'charts' in the name",
			path: "charts/sub-charts-chart/charts/subsub-charts-chart",
			want: []string{"", "sub-charts-chart", "subsub-charts-chart"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pathToCharts(tt.path); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("pathToCharts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_shouldMapUpstreamPath(t *testing.T) {
	type args struct {
		upstreamPath string
	}
	tests := []struct {
		name         string
		upstreamPath string
		want         bool
	}{
		{
			name:         "parent chart",
			upstreamPath: "Chart.yaml",
			want:         true,
		},
		{
			name:         "subchart under 'charts' dir",
			upstreamPath: "charts/subchart/Chart.yaml",
			want:         true,
		},
		{
			name:         "subsubchart under 'charts' dir",
			upstreamPath: "charts/subchart/charts/subsubchart/Chart.yaml",
			want:         true,
		},
		{
			name:         "subchart NOT under 'charts' dir",
			upstreamPath: "subcharts/subchart/Chart.yaml",
			want:         false,
		},
		{
			name:         "subsubchart NOT under 'charts' dir",
			upstreamPath: "subcharts/subchart/subcharts/subsubchart/Chart.yaml",
			want:         false,
		},
		{
			name:         "some random file that ends with Chart.yaml",
			upstreamPath: "charts/subchart/MyChart.yaml",
			want:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldMapUpstreamPath(tt.upstreamPath); got != tt.want {
				t.Errorf("shouldMapUpstreamPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getUpstreamToBasePathsMap(t *testing.T) {
	tests := []struct {
		name          string
		upstreamFiles map[string][]byte
		want          map[string][]string
	}{
		{
			name: "subsubchart with no aliased dependencies",
			upstreamFiles: map[string][]byte{
				"Chart.yaml": []byte(`dependencies:
- name: subchart
  repository: file://./charts/subchart
  version: 0.0.0`),
				"charts/subchart/Chart.yaml": []byte(`dependencies:
- name: subsubchart
  repository: file://./charts/subsubchart
  version: 0.0.0`),
				"charts/subchart/charts/subsubchart/Chart.yaml": []byte(``),
			},
			want: map[string][]string{
				"": {""},
				"charts/subchart": {
					"charts/subchart",
				},
				"charts/subchart/charts/subsubchart": {
					"charts/subchart/charts/subsubchart",
				},
			},
		},
		{
			name: "subsubchart with and without aliased dependencies",
			upstreamFiles: map[string][]byte{
				"Chart.yaml": []byte(`dependencies:
- name: subchart
  repository: file://./charts/subchart
  version: 0.0.0
- name: subchart
  alias: subchart-aliased
  repository: file://./charts/subchart
  version: 0.0.0`),
				"charts/subchart/Chart.yaml": []byte(`dependencies:
- name: subsubchart
  repository: file://./charts/subsubchart
  version: 0.0.0
- name: subsubchart
  alias: subsubchart-aliased
  repository: file://./charts/subsubchart
  version: 0.0.0`),
				"charts/subchart/charts/subsubchart/Chart.yaml": []byte(``),
			},
			want: map[string][]string{
				"": {""},
				"charts/subchart": {
					"charts/subchart",
					"charts/subchart-aliased",
				},
				"charts/subchart/charts/subsubchart": {
					"charts/subchart/charts/subsubchart",
					"charts/subchart/charts/subsubchart-aliased",
					"charts/subchart-aliased/charts/subsubchart",
					"charts/subchart-aliased/charts/subsubchart-aliased",
				},
			},
		},
		{
			name: "subsubchart with only aliased dependencies",
			upstreamFiles: map[string][]byte{
				"Chart.yaml": []byte(`dependencies:
- name: subchart
  alias: subchart-aliased-1
  repository: file://./charts/subchart
  version: 0.0.0
- name: subchart
  alias: subchart-aliased-2
  repository: file://./charts/subchart
  version: 0.0.0`),
				"charts/subchart/Chart.yaml": []byte(`dependencies:
- name: subsubchart
  alias: subsubchart-aliased-1
  repository: file://./charts/subsubchart
  version: 0.0.0
- name: subsubchart
  alias: subsubchart-aliased-2
  repository: file://./charts/subsubchart
  version: 0.0.0`),
				"charts/subchart/charts/subsubchart/Chart.yaml": []byte(``),
			},
			want: map[string][]string{
				"": {""},
				"charts/subchart": {
					"charts/subchart-aliased-1",
					"charts/subchart-aliased-2",
				},
				"charts/subchart/charts/subsubchart": {
					"charts/subchart-aliased-1/charts/subsubchart-aliased-1",
					"charts/subchart-aliased-1/charts/subsubchart-aliased-2",
					"charts/subchart-aliased-2/charts/subsubchart-aliased-1",
					"charts/subchart-aliased-2/charts/subsubchart-aliased-2",
				},
			},
		},
		{
			name: "subsubchart with unlisted dependencies",
			upstreamFiles: map[string][]byte{
				"Chart.yaml": []byte(`dependencies:
- name: subchart
  repository: file://./charts/subchart
  version: 0.0.0`),
				"charts/subchart/Chart.yaml": []byte(`dependencies:
- name: subsubchart
  repository: file://./charts/subsubchart
  version: 0.0.0`),
				"charts/subchart-unlisted/Chart.yaml": []byte(`dependencies:
- name: subsubchart
  alias: subsubchart-aliased
  repository: file://./charts/subsubchart
  version: 0.0.0`),
				"charts/subchart/charts/subsubchart/Chart.yaml":                   []byte(``),
				"charts/subchart-unlisted/charts/subsubchart/Chart.yaml":          []byte(``),
				"charts/subchart-unlisted/charts/subsubchart-unlisted/Chart.yaml": []byte(``),
			},
			want: map[string][]string{
				"": {""},
				"charts/subchart": {
					"charts/subchart",
				},
				"charts/subchart-unlisted": {
					"charts/subchart-unlisted",
				},
				"charts/subchart/charts/subsubchart": {
					"charts/subchart/charts/subsubchart",
				},
				"charts/subchart-unlisted/charts/subsubchart": {
					"charts/subchart-unlisted/charts/subsubchart-aliased",
				},
				"charts/subchart-unlisted/charts/subsubchart-unlisted": {
					"charts/subchart-unlisted/charts/subsubchart-unlisted",
				},
			},
		},
		{
			name: "subsubsubchart with and without aliased dependencies",
			upstreamFiles: map[string][]byte{
				"Chart.yaml": []byte(`dependencies:
- name: subchart
  repository: file://./charts/subchart
  version: 0.0.0
- name: subchart
  alias: subchart-aliased
  repository: file://./charts/subchart
  version: 0.0.0`),
				"charts/subchart/Chart.yaml": []byte(`dependencies:
- name: subsubchart
  repository: file://./charts/subsubchart
  version: 0.0.0
- name: subsubchart
  alias: subsubchart-aliased
  repository: file://./charts/subsubchart
  version: 0.0.0`),
				"charts/subchart/charts/subsubchart/Chart.yaml": []byte(`dependencies:
- name: subsubsubchart
  repository: file://./charts/subsubsubchart
  version: 0.0.0
- name: subsubsubchart
  alias: subsubsubchart-aliased
  repository: file://./charts/subsubsubchart
  version: 0.0.0`),
				"charts/subchart/charts/subsubchart/charts/subsubsubchart/Chart.yaml": []byte(``),
			},
			want: map[string][]string{
				"": {""},
				"charts/subchart": {
					"charts/subchart",
					"charts/subchart-aliased",
				},
				"charts/subchart/charts/subsubchart": {
					"charts/subchart/charts/subsubchart",
					"charts/subchart/charts/subsubchart-aliased",
					"charts/subchart-aliased/charts/subsubchart",
					"charts/subchart-aliased/charts/subsubchart-aliased",
				},
				"charts/subchart/charts/subsubchart/charts/subsubsubchart": {
					"charts/subchart/charts/subsubchart/charts/subsubsubchart",
					"charts/subchart/charts/subsubchart/charts/subsubsubchart-aliased",
					"charts/subchart/charts/subsubchart-aliased/charts/subsubsubchart",
					"charts/subchart/charts/subsubchart-aliased/charts/subsubsubchart-aliased",
					"charts/subchart-aliased/charts/subsubchart/charts/subsubsubchart",
					"charts/subchart-aliased/charts/subsubchart/charts/subsubsubchart-aliased",
					"charts/subchart-aliased/charts/subsubchart-aliased/charts/subsubsubchart",
					"charts/subchart-aliased/charts/subsubchart-aliased/charts/subsubsubchart-aliased",
				},
			},
		},
		{
			name: "ignores subcharts not under a 'charts' directory",
			upstreamFiles: map[string][]byte{
				"Chart.yaml": []byte(`dependencies:
- name: subchart
  repository: file://./subcharts/subchart
  version: 0.0.0`),
				"subcharts/subchart/Chart.yaml": []byte(`dependencies:
- name: subsubchart
  repository: file://./subcharts/subsubchart
  version: 0.0.0`),
				"subcharts/subchart/subcharts/subsubchart/Chart.yaml": []byte(``),
			},
			want: map[string][]string{
				"": {""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getUpstreamToBasePathsMap(tt.upstreamFiles); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getUpstreamToBasePathsMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
