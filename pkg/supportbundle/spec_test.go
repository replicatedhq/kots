package supportbundle

import (
	"reflect"
	"testing"

	"github.com/replicatedhq/kots/pkg/util"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/stretchr/testify/require"
)

func TestBuilder_populateNamespaces(t *testing.T) {
	origPodNamespace := util.PodNamespace
	util.PodNamespace = "populateNamespaces"
	defer func() {
		util.PodNamespace = origPodNamespace
	}()

	tests := []struct {
		name                  string
		minimalRBACNamespaces []string
		supportBundle         *troubleshootv1beta2.SupportBundle
		want                  *troubleshootv1beta2.SupportBundle
	}{
		{
			name:                  "all",
			minimalRBACNamespaces: []string{},
			supportBundle: &troubleshootv1beta2.SupportBundle{
				Spec: troubleshootv1beta2.SupportBundleSpec{
					Collectors: []*troubleshootv1beta2.Collect{
						{
							Secret: &troubleshootv1beta2.Secret{
								Namespace: `repl{{ ConfigOption "test" }}`,
							},
							Run: &troubleshootv1beta2.Run{
								Namespace: util.PodNamespace,
							},
							Logs: &troubleshootv1beta2.Logs{
								Namespace: "hardcoded",
							},
							Exec: &troubleshootv1beta2.Exec{
								Namespace: "",
							},
							Copy: &troubleshootv1beta2.Copy{
								Namespace: `repl{{ Namespace }}`,
							},
							ClusterResources: &troubleshootv1beta2.ClusterResources{},
						},
					},
				},
			},
			want: &troubleshootv1beta2.SupportBundle{
				Spec: troubleshootv1beta2.SupportBundleSpec{
					Collectors: []*troubleshootv1beta2.Collect{
						{
							Secret: &troubleshootv1beta2.Secret{
								Namespace: `repl{{ ConfigOption "test" }}`,
							},
							Run: &troubleshootv1beta2.Run{
								Namespace: util.PodNamespace,
							},
							Logs: &troubleshootv1beta2.Logs{
								Namespace: "hardcoded",
							},
							Exec: &troubleshootv1beta2.Exec{
								Namespace: util.PodNamespace,
							},
							Copy: &troubleshootv1beta2.Copy{
								Namespace: `repl{{ Namespace }}`,
							},
							ClusterResources: &troubleshootv1beta2.ClusterResources{}, // we do not inject a single namespace for the ClusterResources collector
						},
					},
				},
			},
		},
		{
			name:                  "minimal rbac namespaces - preserve",
			minimalRBACNamespaces: []string{"rbac-namespace-1", "rbac-namespace-2"},
			supportBundle: &troubleshootv1beta2.SupportBundle{
				Spec: troubleshootv1beta2.SupportBundleSpec{
					Collectors: []*troubleshootv1beta2.Collect{
						{
							ClusterResources: &troubleshootv1beta2.ClusterResources{
								Namespaces: []string{"preserve-me", "and-me"},
							},
						},
					},
				},
			},
			want: &troubleshootv1beta2.SupportBundle{
				Spec: troubleshootv1beta2.SupportBundleSpec{
					Collectors: []*troubleshootv1beta2.Collect{
						{
							ClusterResources: &troubleshootv1beta2.ClusterResources{
								Namespaces: []string{"preserve-me", "and-me"},
							},
						},
					},
				},
			},
		},
		{
			name:                  "minimal rbac namespaces - override",
			minimalRBACNamespaces: []string{"rbac-namespace-1", "rbac-namespace-2"},
			supportBundle: &troubleshootv1beta2.SupportBundle{
				Spec: troubleshootv1beta2.SupportBundleSpec{
					Collectors: []*troubleshootv1beta2.Collect{
						{
							ClusterResources: &troubleshootv1beta2.ClusterResources{},
						},
					},
				},
			},
			want: &troubleshootv1beta2.SupportBundle{
				Spec: troubleshootv1beta2.SupportBundleSpec{
					Collectors: []*troubleshootv1beta2.Collect{
						{
							ClusterResources: &troubleshootv1beta2.ClusterResources{
								Namespaces: []string{"rbac-namespace-1", "rbac-namespace-2"},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			got := populateNamespaces(tt.supportBundle, tt.minimalRBACNamespaces)

			req.Equal(tt.want, got)
		})
	}
}

func Test_deduplicatedCollectors(t *testing.T) {
	type args struct {
		supportBundle *troubleshootv1beta2.SupportBundle
	}
	tests := []struct {
		name string
		args args
		want *troubleshootv1beta2.SupportBundle
	}{
		{
			name: "basic",
			args: args{
				supportBundle: &troubleshootv1beta2.SupportBundle{
					Spec: troubleshootv1beta2.SupportBundleSpec{
						Collectors: []*troubleshootv1beta2.Collect{
							{
								ClusterResources: &troubleshootv1beta2.ClusterResources{
									CollectorMeta: troubleshootv1beta2.CollectorMeta{CollectorName: "first"},
								},
							},
							{
								ClusterInfo: &troubleshootv1beta2.ClusterInfo{
									CollectorMeta: troubleshootv1beta2.CollectorMeta{CollectorName: "first"},
								},
							},
							{
								ClusterResources: &troubleshootv1beta2.ClusterResources{},
							},
							{
								ClusterInfo: &troubleshootv1beta2.ClusterInfo{},
							},
						},
					},
				},
			},
			want: &troubleshootv1beta2.SupportBundle{
				Spec: troubleshootv1beta2.SupportBundleSpec{
					Collectors: []*troubleshootv1beta2.Collect{
						{
							ClusterResources: &troubleshootv1beta2.ClusterResources{
								CollectorMeta: troubleshootv1beta2.CollectorMeta{CollectorName: "first"},
							},
						},
						{
							ClusterInfo: &troubleshootv1beta2.ClusterInfo{
								CollectorMeta: troubleshootv1beta2.CollectorMeta{CollectorName: "first"},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deduplicatedCollectors(tt.args.supportBundle); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("deduplicatedCollectors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deduplicatedAnalyzers(t *testing.T) {
	type args struct {
		supportBundle *troubleshootv1beta2.SupportBundle
	}
	tests := []struct {
		name string
		args args
		want *troubleshootv1beta2.SupportBundle
	}{
		{
			name: "basic",
			args: args{
				supportBundle: &troubleshootv1beta2.SupportBundle{
					Spec: troubleshootv1beta2.SupportBundleSpec{
						Analyzers: []*troubleshootv1beta2.Analyze{
							{
								ClusterVersion: &troubleshootv1beta2.ClusterVersion{
									AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{CheckName: "first"},
								},
							},
							{
								Longhorn: &troubleshootv1beta2.LonghornAnalyze{
									AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{CheckName: "first"},
								},
							},
							{
								ClusterVersion: &troubleshootv1beta2.ClusterVersion{},
							},
							{
								Longhorn: &troubleshootv1beta2.LonghornAnalyze{},
							},
						},
					},
				},
			},
			want: &troubleshootv1beta2.SupportBundle{
				Spec: troubleshootv1beta2.SupportBundleSpec{
					Analyzers: []*troubleshootv1beta2.Analyze{
						{
							ClusterVersion: &troubleshootv1beta2.ClusterVersion{
								AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{CheckName: "first"},
							},
						},
						{
							Longhorn: &troubleshootv1beta2.LonghornAnalyze{
								AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{CheckName: "first"},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deduplicatedAnalyzers(tt.args.supportBundle); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("deduplicatedAnalyzers() = %v, want %v", got, tt.want)
			}
		})
	}
}
