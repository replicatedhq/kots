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

func unstructuredWithAnnotation(key, value string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{"metadata": map[string]interface{}{"annotations": map[string]interface{}{key: value}}},
	}
}

func TestResources_GroupByPhaseAnnotation(t *testing.T) {
	tests := []struct {
		name       string
		r          Resources
		annotation string
		want       map[string]Resources
	}{
		{
			name:       "no resources",
			r:          Resources{},
			annotation: CreationPhaseAnnotation,
			want:       map[string]Resources{},
		},
		{
			name: "no annotation",
			r: Resources{
				{Unstructured: &unstructured.Unstructured{}},
			},
			annotation: CreationPhaseAnnotation,
			want: map[string]Resources{
				"0": {
					{Unstructured: &unstructured.Unstructured{}},
				},
			},
		},
		{
			name: "invalid annotation",
			r: Resources{
				{Unstructured: unstructuredWithAnnotation(CreationPhaseAnnotation, "invalid")},
			},
			annotation: CreationPhaseAnnotation,
			want: map[string]Resources{
				"0": {
					{Unstructured: unstructuredWithAnnotation(CreationPhaseAnnotation, "invalid")},
				},
			},
		},
		{
			name: "has annotation",
			r: Resources{
				{Unstructured: unstructuredWithAnnotation(CreationPhaseAnnotation, "1")},
			},
			annotation: CreationPhaseAnnotation,
			want: map[string]Resources{
				"1": {
					{Unstructured: unstructuredWithAnnotation(CreationPhaseAnnotation, "1")},
				},
			},
		},
		{
			name: "multiple resources with same annotation",
			r: Resources{
				{Unstructured: unstructuredWithAnnotation(CreationPhaseAnnotation, "1")},
				{Unstructured: unstructuredWithAnnotation(CreationPhaseAnnotation, "1")},
			},
			annotation: CreationPhaseAnnotation,
			want: map[string]Resources{
				"1": {
					{Unstructured: unstructuredWithAnnotation(CreationPhaseAnnotation, "1")},
					{Unstructured: unstructuredWithAnnotation(CreationPhaseAnnotation, "1")},
				},
			},
		},
		{
			name: "multiple resources with different annotations",
			r: Resources{
				{Unstructured: unstructuredWithAnnotation(CreationPhaseAnnotation, "1")},
				{Unstructured: unstructuredWithAnnotation(CreationPhaseAnnotation, "2")},
			},
			annotation: CreationPhaseAnnotation,
			want: map[string]Resources{
				"1": {
					{Unstructured: unstructuredWithAnnotation(CreationPhaseAnnotation, "1")},
				},
				"2": {
					{Unstructured: unstructuredWithAnnotation(CreationPhaseAnnotation, "2")},
				},
			},
		},
		{
			name: "multiple resources with different annotations, some without, some with invalid annotations",
			r: Resources{
				{Unstructured: unstructuredWithAnnotation(CreationPhaseAnnotation, "1")},
				{Unstructured: unstructuredWithAnnotation(CreationPhaseAnnotation, "2")},
				{Unstructured: &unstructured.Unstructured{}},
				{Unstructured: unstructuredWithAnnotation(CreationPhaseAnnotation, "invalid")},
			},
			annotation: CreationPhaseAnnotation,
			want: map[string]Resources{
				"1": {
					{Unstructured: unstructuredWithAnnotation(CreationPhaseAnnotation, "1")},
				},
				"2": {
					{Unstructured: unstructuredWithAnnotation(CreationPhaseAnnotation, "2")},
				},
				"0": {
					{Unstructured: &unstructured.Unstructured{}},
					{Unstructured: unstructuredWithAnnotation(CreationPhaseAnnotation, "invalid")},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GroupByPhaseAnnotation(tt.annotation); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resources.GroupByPhaseAnnotation() = %v, want %v", got, tt.want)
			}
		})
	}
}
