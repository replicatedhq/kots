package client

import (
	"reflect"
	"testing"

	"github.com/replicatedhq/kots/pkg/operator/applier"
	"github.com/replicatedhq/kots/pkg/operator/types"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Test_decodeManifests(t *testing.T) {
	type args struct {
		manifests []string
	}
	tests := []struct {
		name string
		args args
		want types.Resources
	}{
		{
			name: "expect no error for valid pod manifest",
			args: args{
				manifests: []string{podManifest},
			},
			want: types.Resources{
				{
					GVK: &schema.GroupVersionKind{
						Group:   "",
						Version: "v1",
						Kind:    "Pod",
					},
					GVR:          schema.GroupVersionResource{},
					Unstructured: unstructuredPod,
				},
			},
		},
		{
			name: "expect no error for invalid pod manifest",
			args: args{
				manifests: []string{`test: false123`},
			},
			want: types.Resources{
				{
					GVK:          nil,
					GVR:          schema.GroupVersionResource{},
					Unstructured: nil,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeManifests(tt.args.manifests)
			if len(got) != len(tt.want) {
				t.Errorf("decodeManifests() got = %v, want %v", len(got), len(tt.want))
			}
			for i := range got {
				if !reflect.DeepEqual(got[i].GVK, tt.want[i].GVK) {
					t.Errorf("decodeManifests() got = %v, want %v", got[i].GVK, tt.want[i].GVK)
				}
				if !reflect.DeepEqual(got[i].GVR, tt.want[i].GVR) {
					t.Errorf("decodeManifests() got = %v, want %v", got[i].GVR, tt.want[i].GVR)
				}
				if !reflect.DeepEqual(got[i].Unstructured, tt.want[i].Unstructured) {
					t.Errorf("decodeManifests() got = %v, want %v", got[i].Unstructured, tt.want[i].Unstructured)
				}
			}
		})
	}
}

func Test_deleteManifests(t *testing.T) {
	type args struct {
		manifests         []string
		targetNS          string
		kubernetesApplier applier.KubectlInterface
		waitFlag          bool
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "deleting empty manifests",
			args: args{
				manifests:         []string{},
				targetNS:          "",
				kubernetesApplier: nil,
				waitFlag:          false,
			},
		},
		{
			name: "deleting manifests",
			args: args{
				manifests:         []string{podManifest, rabbitmqCRManifest},
				targetNS:          "test",
				kubernetesApplier: &kubectlApplierMock,
				waitFlag:          false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteManifests(tt.args.manifests, tt.args.targetNS, tt.args.kubernetesApplier, tt.args.waitFlag)
		})
	}
}

func Test_deleteResource(t *testing.T) {
	gvk := schema.GroupVersionKind{
		Group:   "group",
		Version: "version",
		Kind:    "kind",
	}
	type args struct {
		resource          types.Resource
		targetNS          string
		waitFlag          bool
		kubernetesApplier applier.KubectlInterface
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "expect no error for resource with GVKN",
			args: args{
				resource: types.Resource{
					GVK:          &gvk,
					Unstructured: unstructuredPodWithLabels,
				},
				targetNS:          "default",
				kubernetesApplier: &kubectlApplierMock,
			},
		}, {
			name: "expect no error for resource without GVKN",
			args: args{
				resource: types.Resource{
					Unstructured: unstructuredPodWithLabels,
				},
				targetNS:          "default",
				kubernetesApplier: &kubectlApplierMock,
			},
		}, {
			name: "expect no error for resource without Unstructured",
			args: args{
				resource: types.Resource{
					GVK: &gvk,
				},
				targetNS:          "default",
				kubernetesApplier: &kubectlApplierMock,
			},
		}, {
			name: "expect no error for resource with Unstructured without namespace",
			args: args{
				resource: types.Resource{
					GVK: &gvk,
					Unstructured: &unstructured.Unstructured{
						Object: map[string]interface{}{},
					},
				},
				targetNS:          "default",
				kubernetesApplier: &kubectlApplierMock,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteResource(tt.args.resource, tt.args.targetNS, tt.args.waitFlag, tt.args.kubernetesApplier)
		})
	}
}

func Test_shouldWaitForResourceDeletion(t *testing.T) {
	type args struct {
		kind     string
		waitFlag bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "expect true when wait flag is true",
			args: args{
				kind:     "Pod",
				waitFlag: true,
			},
			want: true,
		}, {
			name: "expect false when wait flag is false",
			args: args{
				kind:     "Pod",
				waitFlag: false,
			},
			want: false,
		}, {
			name: "expect false when kind is PersistentVolumeClaim",
			args: args{
				kind:     "PersistentVolumeClaim",
				waitFlag: true,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldWaitForResourceDeletion(tt.args.kind, tt.args.waitFlag); got != tt.want {
				t.Errorf("shouldWaitForResourceDeletion() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
