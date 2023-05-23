package types

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestResource_ShouldWaitForReady(t *testing.T) {
	tests := []struct {
		name     string
		resource Resource
		want     bool
	}{
		{
			name: "no annotations",
			resource: Resource{
				Unstructured: &unstructured.Unstructured{},
			},
			want: false,
		},
		{
			name: "no wait-for-ready annotation",
			resource: Resource{
				Unstructured: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{
							"annotations": map[string]interface{}{
								"kots.io/something-else": "true",
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "wait-for-ready annotation not true",
			resource: Resource{
				Unstructured: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{
							"annotations": map[string]interface{}{
								"kots.io/wait-for-ready": "false",
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "wait-for-ready annotation true",
			resource: Resource{
				Unstructured: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{
							"annotations": map[string]interface{}{
								"kots.io/wait-for-ready": "true",
							},
						},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.resource.ShouldWaitForReady(); got != tt.want {
				t.Errorf("Resource.ShouldWaitForReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResource_ShouldWaitForProperties(t *testing.T) {
	tests := []struct {
		name     string
		resource Resource
		want     bool
	}{
		{
			name: "no annotations",
			resource: Resource{
				Unstructured: &unstructured.Unstructured{},
			},
			want: false,
		},
		{
			name: "no wait-for-properties annotation",
			resource: Resource{
				Unstructured: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{
							"annotations": map[string]interface{}{
								"kots.io/something-else": "true",
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "has wait-for-properties annotation",
			resource: Resource{
				Unstructured: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{
							"annotations": map[string]interface{}{
								"kots.io/wait-for-properties": ".status.ready=true,.status.something-else=1",
							},
						},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.resource.ShouldWaitForProperties(); got != tt.want {
				t.Errorf("Resource.ShouldWaitForProperties() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResource_GetWaitForProperties(t *testing.T) {
	tests := []struct {
		name     string
		resource Resource
		want     []WaitForProperty
	}{
		{
			name: "no annotations",
			resource: Resource{
				Unstructured: &unstructured.Unstructured{},
			},
			want: nil,
		},
		{
			name: "no wait-for-properties annotation",
			resource: Resource{
				Unstructured: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{
							"annotations": map[string]interface{}{
								"kots.io/something-else": "true",
							},
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "has wait-for-properties annotation",
			resource: Resource{
				Unstructured: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{
							"annotations": map[string]interface{}{
								"kots.io/wait-for-properties": ".status.ready=true,.status.something-else=1",
							},
						},
					},
				},
			},
			want: []WaitForProperty{
				{
					Path:  ".status.ready",
					Value: "true",
				},
				{
					Path:  ".status.something-else",
					Value: "1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.resource.GetWaitForProperties(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resource.GetWaitForProperties() = %v, want %v", got, tt.want)
			}
		})
	}
}
