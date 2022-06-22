package types

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestResourceRequirements_ToCoreV1ResourceRequirements(t *testing.T) {
	tests := []struct {
		name string
		r    *ResourceRequirements
		want corev1.ResourceRequirements
	}{
		{
			name: "nil",
			want: corev1.ResourceRequirements{},
		},
		{
			name: "empty",
			r:    &ResourceRequirements{},
			want: corev1.ResourceRequirements{},
		},
		{
			name: "basic",
			r: &ResourceRequirements{
				PodCpuRequest:         resource.MustParse("1"),
				PodCpuRequestIsSet:    true,
				PodMemoryRequest:      resource.MustParse("2"),
				PodMemoryRequestIsSet: true,
				PodCpuLimit:           resource.MustParse("3"),
				PodCpuLimitIsSet:      true,
				PodMemoryLimit:        resource.MustParse("4"),
				PodMemoryLimitIsSet:   true,
			},
			want: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("1"),
					"memory": resource.MustParse("2"),
				},
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("3"),
					"memory": resource.MustParse("4"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.ToCoreV1ResourceRequirements(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResourceRequirements.ToCoreV1ResourceRequirements() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResourceRequirements_UpdateCoreV1ResourceRequirements(t *testing.T) {
	tests := []struct {
		name      string
		r         *ResourceRequirements
		resources corev1.ResourceRequirements
		want      corev1.ResourceRequirements
	}{
		{
			name:      "nil",
			resources: corev1.ResourceRequirements{},
			want:      corev1.ResourceRequirements{},
		},
		{
			name: "empty",
			r:    &ResourceRequirements{},
			resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("1"),
					"memory": resource.MustParse("2"),
				},
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("3"),
					"memory": resource.MustParse("4"),
				},
			},
			want: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("1"),
					"memory": resource.MustParse("2"),
				},
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("3"),
					"memory": resource.MustParse("4"),
				},
			},
		},
		{
			name: "basic",
			r: &ResourceRequirements{
				PodCpuRequest:         resource.MustParse("1"),
				PodCpuRequestIsSet:    true,
				PodMemoryRequest:      resource.MustParse("2"),
				PodMemoryRequestIsSet: true,
				PodCpuLimit:           resource.MustParse("3"),
				PodCpuLimitIsSet:      true,
				PodMemoryLimit:        resource.MustParse("4"),
				PodMemoryLimitIsSet:   true,
			},
			resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("5"),
					"memory": resource.MustParse("6"),
				},
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("7"),
					"memory": resource.MustParse("8"),
				},
			},
			want: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("1"),
					"memory": resource.MustParse("2"),
				},
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("3"),
					"memory": resource.MustParse("4"),
				},
			},
		},
		{
			name: "partial",
			r: &ResourceRequirements{
				PodCpuRequest:       resource.MustParse("1"),
				PodCpuRequestIsSet:  true,
				PodMemoryLimit:      resource.MustParse("4"),
				PodMemoryLimitIsSet: true,
			},
			resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("5"),
					"memory": resource.MustParse("6"),
				},
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("7"),
					"memory": resource.MustParse("8"),
				},
			},
			want: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("1"),
					"memory": resource.MustParse("6"),
				},
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("7"),
					"memory": resource.MustParse("4"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.UpdateCoreV1ResourceRequirements(tt.resources); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResourceRequirements.UpdateCoreV1ResourceRequirements() = %v, want %v", got, tt.want)
			}
		})
	}
}
