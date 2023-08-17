package client

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/replicatedhq/kots/pkg/operator/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestDecodeManifests(t *testing.T) {
	tests := []struct {
		name     string
		input    [][]byte
		expected types.Resources
	}{
		{
			name: "decodes valid manifest",
			input: [][]byte{[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: example
spec:
  containers:
  - name: example
    image: ubuntu:16.04
`)},
			expected: types.Resources{
				{
					Manifest: `apiVersion: v1
kind: Pod
metadata:
  name: example
spec:
  containers:
  - name: example
    image: ubuntu:16.04
`,
					Unstructured: &unstructured.Unstructured{
						Object: map[string]interface{}{
							"apiVersion": "v1",
							"kind":       "Pod",
							"metadata": map[string]interface{}{
								"name": "example",
							},
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name":  "example",
										"image": "ubuntu:16.04",
									},
								},
							},
						},
					},
					GVK: &schema.GroupVersionKind{
						Group:   "",
						Version: "v1",
						Kind:    "Pod",
					},
				},
			},
		},
		{
			name:  "saves error for invalid manifest",
			input: [][]byte{[]byte(`invalid manifest`)},
			expected: types.Resources{
				{
					Manifest:     `invalid manifest`,
					DecodeErrMsg: `couldn't get version/kind; json parse error: json: cannot unmarshal string into Go value of type struct { APIVersion string "json:\"apiVersion,omitempty\""; Kind string "json:\"kind,omitempty\"" }`,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := decodeManifests(test.input)
			b, _ := json.MarshalIndent(actual, "", "  ")
			t.Log(string(b))
			if !reflect.DeepEqual(actual, test.expected) {
				t.Errorf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}

func unstructuredWithAnnotation(key, value string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{"metadata": map[string]interface{}{"annotations": map[string]interface{}{key: value}}},
	}
}

func Test_groupAndSortResourcesForCreation(t *testing.T) {
	tests := []struct {
		name     string
		input    types.Resources
		expected types.Phases
	}{
		{
			name: "sorts known kinds",
			input: types.Resources{
				{GVK: &schema.GroupVersionKind{Kind: "PodSecurityPolicy"}},
				{GVK: &schema.GroupVersionKind{Kind: "LimitRange"}},
				{GVK: &schema.GroupVersionKind{Kind: "ResourceQuota"}},
				{GVK: &schema.GroupVersionKind{Kind: "Namespace"}},
				{GVK: &schema.GroupVersionKind{Kind: "PodDisruptionBudget"}},
				{GVK: &schema.GroupVersionKind{Kind: "Secret"}},
				{GVK: &schema.GroupVersionKind{Kind: "ServiceAccount"}},
				{GVK: &schema.GroupVersionKind{Kind: "SecretList"}},
				{GVK: &schema.GroupVersionKind{Kind: "ConfigMap"}},
				{GVK: &schema.GroupVersionKind{Kind: "PersistentVolume"}},
				{GVK: &schema.GroupVersionKind{Kind: "ClusterRoleBindingList"}},
				{GVK: &schema.GroupVersionKind{Kind: "PersistentVolumeClaim"}},
				{GVK: &schema.GroupVersionKind{Kind: "NetworkPolicy"}},
				{GVK: &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"}},
				{GVK: &schema.GroupVersionKind{Kind: "RoleBinding"}},
				{GVK: &schema.GroupVersionKind{Kind: "CustomResourceDefinition"}},
				{GVK: &schema.GroupVersionKind{Kind: "ClusterRoleList"}},
				{GVK: &schema.GroupVersionKind{Kind: "StorageClass"}},
				{GVK: &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v1"}},
				{GVK: &schema.GroupVersionKind{Kind: "Role"}},
				{GVK: &schema.GroupVersionKind{Kind: "ClusterRole"}},
				{
					GVK:          &schema.GroupVersionKind{Kind: "Job"},
					Unstructured: unstructuredWithAnnotation(types.CreationPhaseAnnotation, "1"),
				},
				{
					GVK:          &schema.GroupVersionKind{Kind: "Job"},
					Unstructured: unstructuredWithAnnotation(types.CreationPhaseAnnotation, "-1"),
				},
				{
					GVK:          &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"},
					Unstructured: unstructuredWithAnnotation(types.CreationPhaseAnnotation, "1"),
				},
				{
					GVK:          &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"},
					Unstructured: unstructuredWithAnnotation(types.CreationPhaseAnnotation, "-1"),
				},
				{
					GVK:          &schema.GroupVersionKind{Kind: "Job"},
					Unstructured: unstructuredWithAnnotation(types.CreationPhaseAnnotation, "2"),
				},
				{
					GVK:          &schema.GroupVersionKind{Kind: "Job"},
					Unstructured: unstructuredWithAnnotation(types.CreationPhaseAnnotation, "-2"),
				},
				{
					GVK:          &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"},
					Unstructured: unstructuredWithAnnotation(types.CreationPhaseAnnotation, "2"),
				},
				{
					GVK:          &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"},
					Unstructured: unstructuredWithAnnotation(types.CreationPhaseAnnotation, "-2"),
				},
				{GVK: &schema.GroupVersionKind{Kind: "RoleList"}},
				{GVK: &schema.GroupVersionKind{Kind: "DaemonSet"}},
				{GVK: &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v2"}},
				{GVK: &schema.GroupVersionKind{Kind: "RoleBindingList"}},
				{GVK: &schema.GroupVersionKind{Kind: "ClusterRoleBinding"}},
				{GVK: &schema.GroupVersionKind{Kind: "ReplicationController"}},
				{GVK: &schema.GroupVersionKind{Kind: "Pod"}},
				{GVK: &schema.GroupVersionKind{Kind: "Deployment"}},
				{GVK: &schema.GroupVersionKind{Kind: "ReplicaSet"}},
				{GVK: &schema.GroupVersionKind{Kind: "Job"}},
				{GVK: &schema.GroupVersionKind{Kind: "HorizontalPodAutoscaler"}},
				{GVK: &schema.GroupVersionKind{Kind: "APIService"}},
				{GVK: &schema.GroupVersionKind{Kind: "StatefulSet"}},
				{GVK: &schema.GroupVersionKind{Kind: "Service"}},
				{GVK: &schema.GroupVersionKind{Kind: "IngressClass"}},
				{GVK: &schema.GroupVersionKind{Kind: "CronJob"}},
				{GVK: &schema.GroupVersionKind{Kind: "Ingress"}},
			},
			expected: types.Phases{
				{
					Name: "-2",
					Resources: types.Resources{
						{
							GVK:          &schema.GroupVersionKind{Kind: "Job"},
							Unstructured: unstructuredWithAnnotation(types.CreationPhaseAnnotation, "-2"),
						},
						{
							GVK:          &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"},
							Unstructured: unstructuredWithAnnotation(types.CreationPhaseAnnotation, "-2"),
						},
					},
				},
				{
					Name: "-1",
					Resources: types.Resources{
						{
							GVK:          &schema.GroupVersionKind{Kind: "Job"},
							Unstructured: unstructuredWithAnnotation(types.CreationPhaseAnnotation, "-1"),
						},
						{
							GVK:          &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"},
							Unstructured: unstructuredWithAnnotation(types.CreationPhaseAnnotation, "-1"),
						},
					},
				},
				{
					Name: "0",
					Resources: types.Resources{
						{GVK: &schema.GroupVersionKind{Kind: "Namespace"}},
						{GVK: &schema.GroupVersionKind{Kind: "NetworkPolicy"}},
						{GVK: &schema.GroupVersionKind{Kind: "ResourceQuota"}},
						{GVK: &schema.GroupVersionKind{Kind: "LimitRange"}},
						{GVK: &schema.GroupVersionKind{Kind: "PodSecurityPolicy"}},
						{GVK: &schema.GroupVersionKind{Kind: "PodDisruptionBudget"}},
						{GVK: &schema.GroupVersionKind{Kind: "ServiceAccount"}},
						{GVK: &schema.GroupVersionKind{Kind: "Secret"}},
						{GVK: &schema.GroupVersionKind{Kind: "SecretList"}},
						{GVK: &schema.GroupVersionKind{Kind: "ConfigMap"}},
						{GVK: &schema.GroupVersionKind{Kind: "StorageClass"}},
						{GVK: &schema.GroupVersionKind{Kind: "PersistentVolume"}},
						{GVK: &schema.GroupVersionKind{Kind: "PersistentVolumeClaim"}},
						{GVK: &schema.GroupVersionKind{Kind: "CustomResourceDefinition"}},
						{GVK: &schema.GroupVersionKind{Kind: "ClusterRole"}},
						{GVK: &schema.GroupVersionKind{Kind: "ClusterRoleList"}},
						{GVK: &schema.GroupVersionKind{Kind: "ClusterRoleBinding"}},
						{GVK: &schema.GroupVersionKind{Kind: "ClusterRoleBindingList"}},
						{GVK: &schema.GroupVersionKind{Kind: "Role"}},
						{GVK: &schema.GroupVersionKind{Kind: "RoleList"}},
						{GVK: &schema.GroupVersionKind{Kind: "RoleBinding"}},
						{GVK: &schema.GroupVersionKind{Kind: "RoleBindingList"}},
						{GVK: &schema.GroupVersionKind{Kind: "Service"}},
						{GVK: &schema.GroupVersionKind{Kind: "DaemonSet"}},
						{GVK: &schema.GroupVersionKind{Kind: "Pod"}},
						{GVK: &schema.GroupVersionKind{Kind: "ReplicationController"}},
						{GVK: &schema.GroupVersionKind{Kind: "ReplicaSet"}},
						{GVK: &schema.GroupVersionKind{Kind: "Deployment"}},
						{GVK: &schema.GroupVersionKind{Kind: "HorizontalPodAutoscaler"}},
						{GVK: &schema.GroupVersionKind{Kind: "StatefulSet"}},
						{GVK: &schema.GroupVersionKind{Kind: "Job"}},
						{GVK: &schema.GroupVersionKind{Kind: "CronJob"}},
						{GVK: &schema.GroupVersionKind{Kind: "IngressClass"}},
						{GVK: &schema.GroupVersionKind{Kind: "Ingress"}},
						{GVK: &schema.GroupVersionKind{Kind: "APIService"}},
						{GVK: &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"}}, // unknown kinds are last, original order is preserved
						{GVK: &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v1"}}, // unknown kinds are last, original order is preserved
						{GVK: &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v2"}}, // unknown kinds are last, original order is preserved
					},
				},
				{
					Name: "1",
					Resources: types.Resources{
						{
							GVK:          &schema.GroupVersionKind{Kind: "Job"},
							Unstructured: unstructuredWithAnnotation(types.CreationPhaseAnnotation, "1"),
						},
						{
							GVK:          &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"},
							Unstructured: unstructuredWithAnnotation(types.CreationPhaseAnnotation, "1"),
						},
					},
				},
				{
					Name: "2",
					Resources: types.Resources{
						{
							GVK:          &schema.GroupVersionKind{Kind: "Job"},
							Unstructured: unstructuredWithAnnotation(types.CreationPhaseAnnotation, "2"),
						},
						{
							GVK:          &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"},
							Unstructured: unstructuredWithAnnotation(types.CreationPhaseAnnotation, "2"),
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := groupAndSortResourcesForCreation(test.input)
			if !reflect.DeepEqual(actual, test.expected) {
				t.Errorf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}

func Test_groupAndSortResourcesForDeletion(t *testing.T) {
	tests := []struct {
		name     string
		input    types.Resources
		expected types.Phases
	}{
		{
			name: "sorts known kinds",
			input: types.Resources{
				{GVK: &schema.GroupVersionKind{Kind: "PodSecurityPolicy"}},
				{GVK: &schema.GroupVersionKind{Kind: "LimitRange"}},
				{GVK: &schema.GroupVersionKind{Kind: "ResourceQuota"}},
				{GVK: &schema.GroupVersionKind{Kind: "Namespace"}},
				{GVK: &schema.GroupVersionKind{Kind: "PodDisruptionBudget"}},
				{GVK: &schema.GroupVersionKind{Kind: "Secret"}},
				{GVK: &schema.GroupVersionKind{Kind: "ServiceAccount"}},
				{GVK: &schema.GroupVersionKind{Kind: "SecretList"}},
				{GVK: &schema.GroupVersionKind{Kind: "ConfigMap"}},
				{GVK: &schema.GroupVersionKind{Kind: "PersistentVolume"}},
				{GVK: &schema.GroupVersionKind{Kind: "ClusterRoleBindingList"}},
				{GVK: &schema.GroupVersionKind{Kind: "PersistentVolumeClaim"}},
				{GVK: &schema.GroupVersionKind{Kind: "NetworkPolicy"}},
				{GVK: &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"}},
				{GVK: &schema.GroupVersionKind{Kind: "RoleBinding"}},
				{GVK: &schema.GroupVersionKind{Kind: "CustomResourceDefinition"}},
				{GVK: &schema.GroupVersionKind{Kind: "ClusterRoleList"}},
				{GVK: &schema.GroupVersionKind{Kind: "StorageClass"}},
				{GVK: &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v1"}},
				{GVK: &schema.GroupVersionKind{Kind: "Role"}},
				{GVK: &schema.GroupVersionKind{Kind: "ClusterRole"}},
				{
					GVK:          &schema.GroupVersionKind{Kind: "Job"},
					Unstructured: unstructuredWithAnnotation(types.DeletionPhaseAnnotation, "1"),
				},
				{
					GVK:          &schema.GroupVersionKind{Kind: "Job"},
					Unstructured: unstructuredWithAnnotation(types.DeletionPhaseAnnotation, "-1"),
				},
				{
					GVK:          &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"},
					Unstructured: unstructuredWithAnnotation(types.DeletionPhaseAnnotation, "1"),
				},
				{
					GVK:          &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"},
					Unstructured: unstructuredWithAnnotation(types.DeletionPhaseAnnotation, "-1"),
				},
				{
					GVK:          &schema.GroupVersionKind{Kind: "Job"},
					Unstructured: unstructuredWithAnnotation(types.DeletionPhaseAnnotation, "2"),
				},
				{
					GVK:          &schema.GroupVersionKind{Kind: "Job"},
					Unstructured: unstructuredWithAnnotation(types.DeletionPhaseAnnotation, "-2"),
				},
				{
					GVK:          &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"},
					Unstructured: unstructuredWithAnnotation(types.DeletionPhaseAnnotation, "2"),
				},
				{
					GVK:          &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"},
					Unstructured: unstructuredWithAnnotation(types.DeletionPhaseAnnotation, "-2"),
				},
				{GVK: &schema.GroupVersionKind{Kind: "RoleList"}},
				{GVK: &schema.GroupVersionKind{Kind: "DaemonSet"}},
				{GVK: &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v2"}},
				{GVK: &schema.GroupVersionKind{Kind: "RoleBindingList"}},
				{GVK: &schema.GroupVersionKind{Kind: "ClusterRoleBinding"}},
				{GVK: &schema.GroupVersionKind{Kind: "ReplicationController"}},
				{GVK: &schema.GroupVersionKind{Kind: "Pod"}},
				{GVK: &schema.GroupVersionKind{Kind: "Deployment"}},
				{GVK: &schema.GroupVersionKind{Kind: "ReplicaSet"}},
				{GVK: &schema.GroupVersionKind{Kind: "Job"}},
				{GVK: &schema.GroupVersionKind{Kind: "HorizontalPodAutoscaler"}},
				{GVK: &schema.GroupVersionKind{Kind: "APIService"}},
				{GVK: &schema.GroupVersionKind{Kind: "StatefulSet"}},
				{GVK: &schema.GroupVersionKind{Kind: "Service"}},
				{GVK: &schema.GroupVersionKind{Kind: "IngressClass"}},
				{GVK: &schema.GroupVersionKind{Kind: "CronJob"}},
				{GVK: &schema.GroupVersionKind{Kind: "Ingress"}},
			},
			expected: types.Phases{
				{
					Name: "-2",
					Resources: types.Resources{
						{
							GVK:          &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"},
							Unstructured: unstructuredWithAnnotation(types.DeletionPhaseAnnotation, "-2"),
						},
						{
							GVK:          &schema.GroupVersionKind{Kind: "Job"},
							Unstructured: unstructuredWithAnnotation(types.DeletionPhaseAnnotation, "-2"),
						},
					},
				},
				{
					Name: "-1",
					Resources: types.Resources{
						{
							GVK:          &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"},
							Unstructured: unstructuredWithAnnotation(types.DeletionPhaseAnnotation, "-1"),
						},
						{
							GVK:          &schema.GroupVersionKind{Kind: "Job"},
							Unstructured: unstructuredWithAnnotation(types.DeletionPhaseAnnotation, "-1"),
						},
					},
				},
				{
					Name: "0",
					Resources: types.Resources{
						{GVK: &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"}}, // unknown kinds are first, original order is preserved
						{GVK: &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v1"}}, // unknown kinds are first, original order is preserved
						{GVK: &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v2"}}, // unknown kinds are first, original order is preserved
						{GVK: &schema.GroupVersionKind{Kind: "APIService"}},
						{GVK: &schema.GroupVersionKind{Kind: "Ingress"}},
						{GVK: &schema.GroupVersionKind{Kind: "IngressClass"}},
						{GVK: &schema.GroupVersionKind{Kind: "Service"}},
						{GVK: &schema.GroupVersionKind{Kind: "CronJob"}},
						{GVK: &schema.GroupVersionKind{Kind: "Job"}},
						{GVK: &schema.GroupVersionKind{Kind: "StatefulSet"}},
						{GVK: &schema.GroupVersionKind{Kind: "HorizontalPodAutoscaler"}},
						{GVK: &schema.GroupVersionKind{Kind: "Deployment"}},
						{GVK: &schema.GroupVersionKind{Kind: "ReplicaSet"}},
						{GVK: &schema.GroupVersionKind{Kind: "ReplicationController"}},
						{GVK: &schema.GroupVersionKind{Kind: "Pod"}},
						{GVK: &schema.GroupVersionKind{Kind: "DaemonSet"}},
						{GVK: &schema.GroupVersionKind{Kind: "RoleBindingList"}},
						{GVK: &schema.GroupVersionKind{Kind: "RoleBinding"}},
						{GVK: &schema.GroupVersionKind{Kind: "RoleList"}},
						{GVK: &schema.GroupVersionKind{Kind: "Role"}},
						{GVK: &schema.GroupVersionKind{Kind: "ClusterRoleBindingList"}},
						{GVK: &schema.GroupVersionKind{Kind: "ClusterRoleBinding"}},
						{GVK: &schema.GroupVersionKind{Kind: "ClusterRoleList"}},
						{GVK: &schema.GroupVersionKind{Kind: "ClusterRole"}},
						{GVK: &schema.GroupVersionKind{Kind: "CustomResourceDefinition"}},
						{GVK: &schema.GroupVersionKind{Kind: "PersistentVolumeClaim"}},
						{GVK: &schema.GroupVersionKind{Kind: "PersistentVolume"}},
						{GVK: &schema.GroupVersionKind{Kind: "StorageClass"}},
						{GVK: &schema.GroupVersionKind{Kind: "ConfigMap"}},
						{GVK: &schema.GroupVersionKind{Kind: "SecretList"}},
						{GVK: &schema.GroupVersionKind{Kind: "Secret"}},
						{GVK: &schema.GroupVersionKind{Kind: "ServiceAccount"}},
						{GVK: &schema.GroupVersionKind{Kind: "PodDisruptionBudget"}},
						{GVK: &schema.GroupVersionKind{Kind: "PodSecurityPolicy"}},
						{GVK: &schema.GroupVersionKind{Kind: "LimitRange"}},
						{GVK: &schema.GroupVersionKind{Kind: "ResourceQuota"}},
						{GVK: &schema.GroupVersionKind{Kind: "NetworkPolicy"}},
						{GVK: &schema.GroupVersionKind{Kind: "Namespace"}},
					},
				},
				{
					Name: "1",
					Resources: types.Resources{
						{
							GVK:          &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"},
							Unstructured: unstructuredWithAnnotation(types.DeletionPhaseAnnotation, "1"),
						},
						{
							GVK:          &schema.GroupVersionKind{Kind: "Job"},
							Unstructured: unstructuredWithAnnotation(types.DeletionPhaseAnnotation, "1"),
						},
					},
				},
				{
					Name: "2",
					Resources: types.Resources{
						{
							GVK:          &schema.GroupVersionKind{Kind: "UnknownKind", Group: "unknown", Version: "v3"},
							Unstructured: unstructuredWithAnnotation(types.DeletionPhaseAnnotation, "2"),
						},
						{
							GVK:          &schema.GroupVersionKind{Kind: "Job"},
							Unstructured: unstructuredWithAnnotation(types.DeletionPhaseAnnotation, "2"),
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := groupAndSortResourcesForDeletion(test.input)
			if !reflect.DeepEqual(actual, test.expected) {
				t.Errorf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}

func Test_getSortedPhases(t *testing.T) {
	tests := []struct {
		name             string
		resourcesByPhase map[string]types.Resources
		want             []string
	}{
		{
			name:             "empty",
			resourcesByPhase: map[string]types.Resources{},
			want:             []string{},
		},
		{
			name: "one phase",
			resourcesByPhase: map[string]types.Resources{
				"0": {},
			},
			want: []string{"0"},
		},
		{
			name: "multiple phases",
			resourcesByPhase: map[string]types.Resources{
				"1":     {},
				"-9999": {},
				"0":     {},
				"-2":    {},
			},
			want: []string{"-9999", "-2", "0", "1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getSortedPhases(tt.resourcesByPhase); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getSortedPhases() = %v, want %v", got, tt.want)
			}
		})
	}
}
