package supportbundle

import (
	"context"
	"encoding/base64"
	"reflect"
	"testing"

	"github.com/replicatedhq/kots/pkg/util"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
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
		{
			name: "Multiple ClusterResources",
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
								ClusterResources: &troubleshootv1beta2.ClusterResources{
									CollectorMeta: troubleshootv1beta2.CollectorMeta{CollectorName: "first"},
								},
							},
							{
								ClusterResources: &troubleshootv1beta2.ClusterResources{},
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
					},
				},
			},
		},
		{
			name: "Multiple ClusterInfo",
			args: args{
				supportBundle: &troubleshootv1beta2.SupportBundle{
					Spec: troubleshootv1beta2.SupportBundleSpec{
						Collectors: []*troubleshootv1beta2.Collect{
							{
								ClusterInfo: &troubleshootv1beta2.ClusterInfo{
									CollectorMeta: troubleshootv1beta2.CollectorMeta{CollectorName: "first"},
								},
							},
							{
								ClusterInfo: &troubleshootv1beta2.ClusterInfo{
									CollectorMeta: troubleshootv1beta2.CollectorMeta{CollectorName: "first"},
								},
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
							ClusterInfo: &troubleshootv1beta2.ClusterInfo{
								CollectorMeta: troubleshootv1beta2.CollectorMeta{CollectorName: "first"},
							},
						},
					},
				},
			},
		},
		{
			name: "Multiple Ceph",
			args: args{
				supportBundle: &troubleshootv1beta2.SupportBundle{
					Spec: troubleshootv1beta2.SupportBundleSpec{
						Collectors: []*troubleshootv1beta2.Collect{
							{
								Ceph: &troubleshootv1beta2.Ceph{
									CollectorMeta: troubleshootv1beta2.CollectorMeta{CollectorName: "first"},
								},
							},
							{
								Ceph: &troubleshootv1beta2.Ceph{
									CollectorMeta: troubleshootv1beta2.CollectorMeta{CollectorName: "first"},
								},
							},
							{
								Ceph: &troubleshootv1beta2.Ceph{},
							},
						},
					},
				},
			},
			want: &troubleshootv1beta2.SupportBundle{
				Spec: troubleshootv1beta2.SupportBundleSpec{
					Collectors: []*troubleshootv1beta2.Collect{
						{
							Ceph: &troubleshootv1beta2.Ceph{
								CollectorMeta: troubleshootv1beta2.CollectorMeta{CollectorName: "first"},
							},
						},
					},
				},
			},
		},
		{
			name: "Multiple Longhorn",
			args: args{
				supportBundle: &troubleshootv1beta2.SupportBundle{
					Spec: troubleshootv1beta2.SupportBundleSpec{
						Collectors: []*troubleshootv1beta2.Collect{
							{
								Longhorn: &troubleshootv1beta2.Longhorn{
									CollectorMeta: troubleshootv1beta2.CollectorMeta{CollectorName: "first"},
								},
							},
							{
								Longhorn: &troubleshootv1beta2.Longhorn{
									CollectorMeta: troubleshootv1beta2.CollectorMeta{CollectorName: "first"},
								},
							},
							{
								Longhorn: &troubleshootv1beta2.Longhorn{},
							},
						},
					},
				},
			},
			want: &troubleshootv1beta2.SupportBundle{
				Spec: troubleshootv1beta2.SupportBundleSpec{
					Collectors: []*troubleshootv1beta2.Collect{
						{
							Longhorn: &troubleshootv1beta2.Longhorn{
								CollectorMeta: troubleshootv1beta2.CollectorMeta{CollectorName: "first"},
							},
						},
					},
				},
			},
		},
		{
			name: "Multiple Sysctl",
			args: args{
				supportBundle: &troubleshootv1beta2.SupportBundle{
					Spec: troubleshootv1beta2.SupportBundleSpec{
						Collectors: []*troubleshootv1beta2.Collect{
							{
								Sysctl: &troubleshootv1beta2.Sysctl{
									CollectorMeta: troubleshootv1beta2.CollectorMeta{CollectorName: "first"},
								},
							},
							{
								Sysctl: &troubleshootv1beta2.Sysctl{
									CollectorMeta: troubleshootv1beta2.CollectorMeta{CollectorName: "first"},
								},
							},
							{
								Sysctl: &troubleshootv1beta2.Sysctl{},
							},
						},
					},
				},
			},
			want: &troubleshootv1beta2.SupportBundle{
				Spec: troubleshootv1beta2.SupportBundleSpec{
					Collectors: []*troubleshootv1beta2.Collect{
						{
							Sysctl: &troubleshootv1beta2.Sysctl{
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
		{
			name: "weave report duplicated",
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
								WeaveReport: &troubleshootv1beta2.WeaveReportAnalyze{
									AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{CheckName: "first"},
								},
							},
							{
								ClusterVersion: &troubleshootv1beta2.ClusterVersion{},
							},
							{
								Longhorn: &troubleshootv1beta2.LonghornAnalyze{},
							},
							{
								WeaveReport: &troubleshootv1beta2.WeaveReportAnalyze{},
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
						{
							WeaveReport: &troubleshootv1beta2.WeaveReportAnalyze{
								AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{CheckName: "first"},
							},
						},
					},
				},
			},
		},
		{
			name: "ClusterVersion duplicated",
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
								ClusterVersion: &troubleshootv1beta2.ClusterVersion{
									AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{CheckName: "first"},
								},
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
					},
				},
			},
		},
		{
			name: "Multiple ClusterVersion duplicated",
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
								ClusterVersion: &troubleshootv1beta2.ClusterVersion{
									AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{CheckName: "first"},
								},
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

func Test_findSupportBundleSecrets(t *testing.T) {
	encode := func(s string) []byte {
		return []byte(base64.StdEncoding.EncodeToString([]byte(s)))
	}

	tests := []struct {
		name    string
		objects []runtime.Object
		want    []string
		wantErr bool
	}{
		{
			name: "support bundle specs from configmaps and secrets",
			objects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm-my-app-supportbundle",
						Namespace: "kotsadm",
						Labels: map[string]string{
							"foo":                  "bar",
							"troubleshoot.sh/kind": "support-bundle",
						},
					},
					Data: map[string][]byte{
						"support-bundle-spec": encode("my-app-spec"),
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster-wide-supportbundle",
						Namespace: "another",
						Labels: map[string]string{
							"foo":                  "bar",
							"troubleshoot.sh/kind": "support-bundle",
						},
					},
					Data: map[string]string{
						"support-bundle-spec": "cluster-wide-spec",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm-another-app-supportbundle",
						Namespace: "kotsadm",
						Labels: map[string]string{
							"foo":                  "bar",
							"troubleshoot.sh/kind": "support-bundle",
						},
					},
					Data: map[string][]byte{
						"support-bundle-spec": encode("another-app-spec"),
					},
				},
			},
			want: []string{"cluster-wide-spec"},
		},
		{
			name: "support bundle specs with wrong data",
			objects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm-my-app-supportbundle",
						Namespace: "kotsadm",
						Labels: map[string]string{
							"troubleshoot.sh/kind": "support-bundle",
						},
					},
					Data: map[string][]byte{
						"wrong-key": encode("my-app-spec"),
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster-wide-supportbundle",
						Namespace: "default",
						Labels: map[string]string{
							"troubleshoot.sh/kind": "support-bundle",
						},
					},
				},
			},
			want: []string{},
		},
		{
			name: "fail to find support bundle secrets",
			objects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "some-secret",
						Namespace: "default",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			clientset := testclient.NewSimpleClientset(tt.objects...)
			createNamespaces(t, clientset, "kotsadm", "default", "another")
			got, err := findSupportBundleSpecs(clientset)
			assert.Equal(t, tt.wantErr, err != nil, "findSupportBundleSecrets() error %v, wantErr %v", err, tt.wantErr)
			require.NotNil(t, got)

			assert.ElementsMatchf(t, got, tt.want, "got %v, want %v", got, tt.want)
		})
	}
}

func createNamespaces(t *testing.T, clientset kubernetes.Interface, namespaces ...string) {
	t.Helper()

	for _, ns := range namespaces {
		_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		}, metav1.CreateOptions{})
		require.NoError(t, err)
	}
}

func Test_mergeSupportBundleSpecs(t *testing.T) {
	testBundle := &troubleshootv1beta2.SupportBundle{
		Spec: troubleshootv1beta2.SupportBundleSpec{
			Collectors: []*troubleshootv1beta2.Collect{
				{
					ClusterResources: &troubleshootv1beta2.ClusterResources{
						CollectorMeta: troubleshootv1beta2.CollectorMeta{CollectorName: "first"},
					},
				},
				{
					ClusterResources: &troubleshootv1beta2.ClusterResources{
						CollectorMeta: troubleshootv1beta2.CollectorMeta{CollectorName: "first"},
					},
				},
				{
					ClusterResources: &troubleshootv1beta2.ClusterResources{},
				},
			},
			Analyzers: []*troubleshootv1beta2.Analyze{
				{
					ClusterVersion: &troubleshootv1beta2.ClusterVersion{
						AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{CheckName: "first"},
					},
				},
				{
					ClusterVersion: &troubleshootv1beta2.ClusterVersion{
						AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{CheckName: "first"},
					},
				},
			},
			AfterCollection: []*troubleshootv1beta2.AfterCollection{},
			HostCollectors: []*troubleshootv1beta2.HostCollect{
				{
					CPU: &troubleshootv1beta2.CPU{},
					Memory: &troubleshootv1beta2.Memory{
						HostCollectorMeta: troubleshootv1beta2.HostCollectorMeta{CollectorName: "first"},
					},
				},
				{
					CPU: &troubleshootv1beta2.CPU{},
					Memory: &troubleshootv1beta2.Memory{
						HostCollectorMeta: troubleshootv1beta2.HostCollectorMeta{CollectorName: "first"},
					},
				},
				{
					CPU: &troubleshootv1beta2.CPU{},
					Memory: &troubleshootv1beta2.Memory{
						HostCollectorMeta: troubleshootv1beta2.HostCollectorMeta{CollectorName: "second"},
					},
				},
			},
			HostAnalyzers: []*troubleshootv1beta2.HostAnalyze{
				{
					CPU: &troubleshootv1beta2.CPUAnalyze{
						AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{CheckName: "first"},
					},
				},
				{
					CPU: &troubleshootv1beta2.CPUAnalyze{
						AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{CheckName: "first"},
					},
				},
			},
		},
	}

	builtBundles := map[string]*troubleshootv1beta2.SupportBundle{
		"first": testBundle,
	}
	merged := mergeSupportBundleSpecs(builtBundles)

	assert.Equal(t, 2, len(merged.Spec.Collectors))
	assert.Equal(t, 1, len(merged.Spec.Analyzers))
	assert.Equal(t, 2, len(merged.Spec.HostCollectors))
	assert.Equal(t, 1, len(merged.Spec.HostAnalyzers))
}
