package supportbundle

import (
	"reflect"
	"testing"

	"github.com/replicatedhq/kots/pkg/util"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestBuilder_populateNamespaces(t *testing.T) {
	origPodNamespace := util.PodNamespace
	util.PodNamespace = "populateNamespaces"
	defer func() {
		util.PodNamespace = origPodNamespace
	}()

	tests := []struct {
		name                string
		namespacesToCollect []string
		namespacesToAnalyze []string
		supportBundle       *troubleshootv1beta2.SupportBundle
		want                *troubleshootv1beta2.SupportBundle
	}{
		{
			name:                "all",
			namespacesToCollect: []string{},
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
			name:                "minimal rbac namespaces - preserve",
			namespacesToCollect: []string{"rbac-namespace-1", "rbac-namespace-2"},
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
			name:                "minimal rbac namespaces - override",
			namespacesToCollect: []string{"rbac-namespace-1", "rbac-namespace-2", "rbac-namespace-3"},
			namespacesToAnalyze: []string{"rbac-namespace-1", "rbac-namespace-2"},
			supportBundle: &troubleshootv1beta2.SupportBundle{
				Spec: troubleshootv1beta2.SupportBundleSpec{
					Collectors: []*troubleshootv1beta2.Collect{
						{
							ClusterResources: &troubleshootv1beta2.ClusterResources{},
						},
					},
					Analyzers: []*troubleshootv1beta2.Analyze{
						// these will be assigned namespaces
						{
							DeploymentStatus: &troubleshootv1beta2.DeploymentStatus{},
						},
						{
							JobStatus: &troubleshootv1beta2.JobStatus{},
						},
						{
							ReplicaSetStatus: &troubleshootv1beta2.ReplicaSetStatus{},
						},
						{
							StatefulsetStatus: &troubleshootv1beta2.StatefulsetStatus{},
						},
						// these will not be assigned namespaces
						{
							DeploymentStatus: &troubleshootv1beta2.DeploymentStatus{
								Namespaces: []string{"different-namespace-1", "different-namespace-2"},
							},
						},
						{
							JobStatus: &troubleshootv1beta2.JobStatus{
								Namespace: "different-namespace-1",
							},
						},
						{
							ReplicaSetStatus: &troubleshootv1beta2.ReplicaSetStatus{
								Namespaces: []string{"different-namespace-1", "different-namespace-2"},
							},
						},
						{
							StatefulsetStatus: &troubleshootv1beta2.StatefulsetStatus{
								Namespace: "different-namespace-1",
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
								Namespaces: []string{"rbac-namespace-1", "rbac-namespace-2", "rbac-namespace-3"},
							},
						},
					},
					Analyzers: []*troubleshootv1beta2.Analyze{
						{
							DeploymentStatus: &troubleshootv1beta2.DeploymentStatus{
								Namespaces: []string{"rbac-namespace-1", "rbac-namespace-2"},
							},
						},
						{
							JobStatus: &troubleshootv1beta2.JobStatus{
								Namespaces: []string{"rbac-namespace-1", "rbac-namespace-2"},
							},
						},
						{
							ReplicaSetStatus: &troubleshootv1beta2.ReplicaSetStatus{
								Namespaces: []string{"rbac-namespace-1", "rbac-namespace-2"},
							},
						},
						{
							StatefulsetStatus: &troubleshootv1beta2.StatefulsetStatus{
								Namespaces: []string{"rbac-namespace-1", "rbac-namespace-2"},
							},
						},
						{
							DeploymentStatus: &troubleshootv1beta2.DeploymentStatus{
								Namespaces: []string{"different-namespace-1", "different-namespace-2"},
							},
						},
						{
							JobStatus: &troubleshootv1beta2.JobStatus{
								Namespace: "different-namespace-1",
							},
						},
						{
							ReplicaSetStatus: &troubleshootv1beta2.ReplicaSetStatus{
								Namespaces: []string{"different-namespace-1", "different-namespace-2"},
							},
						},
						{
							StatefulsetStatus: &troubleshootv1beta2.StatefulsetStatus{
								Namespace: "different-namespace-1",
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

			got := populateNamespaces(tt.supportBundle, tt.namespacesToCollect, tt.namespacesToAnalyze)

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

func Test_getSpecSecretsMatchingLabel(t *testing.T) {
	tests := []struct {
		name            string
		secret          []runtime.Object
		targetNamespace string
		targetLabelKey  string
		key             string
		expectSuccess   bool
	}{
		{
			name: "get secret with matching label",
			secret: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-1",
						Namespace: "default",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
					Data: map[string][]byte{
						"spec": []byte("spec-1"),
					},
				},
			},
			key:             "spec",
			targetNamespace: "default",
			targetLabelKey:  "foo=bar",
			expectSuccess:   true,
		},
		{
			name: "get support bundle spec secret with matching label",
			secret: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-2",
						Namespace: "default",
						Labels: map[string]string{
							"troubleshoot.io/kind": "support-bundle",
						},
					},
					Data: map[string][]byte{
						"support-bundle-spec": []byte("spec-1"),
					},
				},
			},
			key:             "support-bundle-spec",
			targetNamespace: "default",
			targetLabelKey:  "troubleshoot.io/kind=support-bundle",
			expectSuccess:   true,
		},
		{
			name: "cannot get support bundle spec secret with wrong spec key",
			secret: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-2",
						Namespace: "default",
						Labels: map[string]string{
							"troubleshoot.io/kind": "support-bundle",
						},
					},
					Data: map[string][]byte{
						"support-bundle-spec": []byte("spec-1"),
					},
				},
			},
			key:             "spec",
			targetNamespace: "default",
			targetLabelKey:  "troubleshoot.io/kind=support-bundle",
			expectSuccess:   false,
		},
		{
			name: "get support bundle spec secret with multiple label selector",
			secret: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-1",
						Namespace: "default",
						Labels: map[string]string{
							"troubleshoot.io/kind": "support-bundle",
						},
					},
					Data: map[string][]byte{
						"support-bundle-spec": []byte("spec-1"),
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-2",
						Namespace: "default",
						Labels: map[string]string{
							"troubleshoot.io/kind": "support-bundle",
						},
					},
					Data: map[string][]byte{
						"support-bundle-spec": []byte("spec-2"),
					},
				},
			},
			key:             "support-bundle-spec",
			targetNamespace: "default",
			targetLabelKey:  "troubleshoot.io/kind=support-bundle",
			expectSuccess:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClientset := fake.NewSimpleClientset(tt.secret...)
			_, err := GetSpecSecretsMatchingLabel(fakeClientset, tt.targetLabelKey, tt.targetNamespace, tt.key)
			if err != nil && tt.expectSuccess {
				t.Errorf("getSpecSecretsMatchingLabel() error = %v, expectSuccess %v", err, tt.expectSuccess)
			} else if err == nil && !tt.expectSuccess {
				t.Errorf("getSpecSecretsMatchingLabel() error = nil, expectSuccess %v", tt.expectSuccess)
			}
		})
	}
}
