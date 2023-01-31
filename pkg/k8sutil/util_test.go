package k8sutil

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestMergeEnvVars(t *testing.T) {
	type args struct {
		desired  []corev1.EnvVar
		existing []corev1.EnvVar
		override bool
	}
	tests := []struct {
		name string
		args args
		want []corev1.EnvVar
	}{
		{
			name: "override",
			args: args{
				desired: []corev1.EnvVar{
					{
						Name:  "FOO",
						Value: "bar",
					},
				},
				existing: []corev1.EnvVar{
					{
						Name:  "FOO",
						Value: "baz",
					},
				},
				override: true,
			},
			want: []corev1.EnvVar{
				{
					Name:  "FOO",
					Value: "bar",
				},
			},
		},
		{
			name: "no override",
			args: args{
				desired: []corev1.EnvVar{
					{
						Name:  "FOO",
						Value: "bar",
					},
				},
				existing: []corev1.EnvVar{
					{
						Name:  "FOO",
						Value: "baz",
					},
				},
				override: false,
			},
			want: []corev1.EnvVar{
				{
					Name:  "FOO",
					Value: "baz",
				},
			},
		},
		{
			name: "add new",
			args: args{
				desired: []corev1.EnvVar{
					{
						Name:  "FOO",
						Value: "bar",
					},
				},
				existing: []corev1.EnvVar{
					{
						Name:  "BAZ",
						Value: "qux",
					},
				},
				override: false,
			},
			want: []corev1.EnvVar{
				{
					Name:  "BAZ",
					Value: "qux",
				},
				{
					Name:  "FOO",
					Value: "bar",
				},
			},
		},
		{
			name: "add new and override",
			args: args{
				desired: []corev1.EnvVar{
					{
						Name:  "FOO",
						Value: "bar",
					},
					{
						Name:  "BAZ",
						Value: "qux",
					},
				},
				existing: []corev1.EnvVar{
					{
						Name:  "BAZ",
						Value: "quux",
					},
				},
				override: true,
			},
			want: []corev1.EnvVar{
				{
					Name:  "BAZ",
					Value: "qux",
				},
				{
					Name:  "FOO",
					Value: "bar",
				},
			},
		},
		{
			name: "add new and no override",
			args: args{
				desired: []corev1.EnvVar{
					{
						Name:  "FOO",
						Value: "bar",
					},
					{
						Name:  "BAZ",
						Value: "qux",
					},
				},
				existing: []corev1.EnvVar{
					{
						Name:  "BAZ",
						Value: "quux",
					},
				},
				override: false,
			},
			want: []corev1.EnvVar{
				{
					Name:  "BAZ",
					Value: "quux",
				},
				{
					Name:  "FOO",
					Value: "bar",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MergeEnvVars(tt.args.desired, tt.args.existing, tt.args.override); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeEnvVars() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeVolumes(t *testing.T) {
	type args struct {
		desired  []corev1.Volume
		existing []corev1.Volume
		override bool
	}
	tests := []struct {
		name string
		args args
		want []corev1.Volume
	}{
		{
			name: "override",
			args: args{
				desired: []corev1.Volume{
					{
						Name: "foo",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
				existing: []corev1.Volume{
					{
						Name: "foo",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{},
						},
					},
				},
				override: true,
			},
			want: []corev1.Volume{
				{
					Name: "foo",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
		{
			name: "no override",
			args: args{
				desired: []corev1.Volume{
					{
						Name: "foo",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
				existing: []corev1.Volume{
					{
						Name: "foo",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{},
						},
					},
				},
				override: false,
			},
			want: []corev1.Volume{
				{
					Name: "foo",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{},
					},
				},
			},
		},
		{
			name: "add new",
			args: args{
				desired: []corev1.Volume{
					{
						Name: "foo",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
				existing: []corev1.Volume{
					{
						Name: "bar",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{},
						},
					},
				},
				override: false,
			},
			want: []corev1.Volume{
				{
					Name: "bar",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{},
					},
				},
				{
					Name: "foo",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
		{
			name: "add new and override",
			args: args{
				desired: []corev1.Volume{
					{
						Name: "foo",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
					{
						Name: "bar",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
				existing: []corev1.Volume{
					{
						Name: "bar",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{},
						},
					},
				},
				override: true,
			},
			want: []corev1.Volume{
				{
					Name: "bar",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "foo",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
		{
			name: "add new and no override",
			args: args{
				desired: []corev1.Volume{
					{
						Name: "foo",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
					{
						Name: "bar",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
				existing: []corev1.Volume{
					{
						Name: "bar",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{},
						},
					},
				},
				override: false,
			},
			want: []corev1.Volume{
				{
					Name: "bar",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{},
					},
				},
				{
					Name: "foo",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MergeVolumes(tt.args.desired, tt.args.existing, tt.args.override); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeVolumes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeVolumeMounts(t *testing.T) {
	type args struct {
		desired  []corev1.VolumeMount
		existing []corev1.VolumeMount
		override bool
	}
	tests := []struct {
		name string
		args args
		want []corev1.VolumeMount
	}{
		{
			name: "override",
			args: args{
				desired: []corev1.VolumeMount{
					{
						Name:      "foo",
						MountPath: "/foo",
					},
				},
				existing: []corev1.VolumeMount{
					{
						Name:      "foo",
						MountPath: "/bar",
					},
				},
				override: true,
			},
			want: []corev1.VolumeMount{
				{
					Name:      "foo",
					MountPath: "/foo",
				},
			},
		},
		{
			name: "no override",
			args: args{
				desired: []corev1.VolumeMount{
					{
						Name:      "foo",
						MountPath: "/foo",
					},
				},
				existing: []corev1.VolumeMount{
					{
						Name:      "foo",
						MountPath: "/bar",
					},
				},
				override: false,
			},
			want: []corev1.VolumeMount{
				{
					Name:      "foo",
					MountPath: "/bar",
				},
			},
		},
		{
			name: "add new and override",
			args: args{
				desired: []corev1.VolumeMount{
					{
						Name:      "foo",
						MountPath: "/foo",
					},
					{
						Name:      "bar",
						MountPath: "/baz",
					},
				},
				existing: []corev1.VolumeMount{
					{
						Name:      "bar",
						MountPath: "/bar",
					},
				},
				override: true,
			},
			want: []corev1.VolumeMount{
				{
					Name:      "bar",
					MountPath: "/baz",
				},
				{
					Name:      "foo",
					MountPath: "/foo",
				},
			},
		},
		{
			name: "add new and no override",
			args: args{
				desired: []corev1.VolumeMount{
					{
						Name:      "foo",
						MountPath: "/foo",
					},
					{
						Name:      "bar",
						MountPath: "/baz",
					},
				},
				existing: []corev1.VolumeMount{
					{
						Name:      "bar",
						MountPath: "/bar",
					},
				},
				override: false,
			},
			want: []corev1.VolumeMount{
				{
					Name:      "bar",
					MountPath: "/bar",
				},
				{
					Name:      "foo",
					MountPath: "/foo",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MergeVolumeMounts(tt.args.desired, tt.args.existing, tt.args.override); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeVolumeMounts() = %v, want %v", got, tt.want)
			}
		})
	}
}
