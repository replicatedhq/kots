package kotsadm

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func Test_MinioStatefulset_InitContainers(t *testing.T) {
	tests := []struct {
		name                             string
		deployOptions                    types.DeployOptions
		size                             resource.Quantity
		wantNumInitContainers            int
		wantNumInitContainerVolumeMounts int
		wantErr                          bool
	}{
		{
			name: "does not add init containers if not migrating",
			deployOptions: types.DeployOptions{
				MigrateToMinioXl: false,
			},
			size:                             resource.MustParse("10Gi"),
			wantNumInitContainers:            0,
			wantNumInitContainerVolumeMounts: 0,
			wantErr:                          false,
		},
		{
			name: "adds init containers if not migrating",
			deployOptions: types.DeployOptions{
				MigrateToMinioXl:  true,
				CurrentMinioImage: "minio/minio:RELEASE.2020-10-27T00-54-19Z",
			},
			size:                             resource.MustParse("10Gi"),
			wantNumInitContainers:            3,
			wantNumInitContainerVolumeMounts: len(append(minioVolumeMounts(), minioXlMigrationVolumeMounts()...)),
			wantErr:                          false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sts, err := MinioStatefulset(tt.deployOptions, tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("MinioStatefulset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(sts.Spec.Template.Spec.InitContainers) != tt.wantNumInitContainers {
				t.Errorf("MinioStatefulset() got = %v, want %v", len(sts.Spec.Template.Spec.InitContainers), tt.wantNumInitContainers)
			}

			if tt.wantNumInitContainers > 0 {
				// check that the init containers are in the right order and have the additional volume mounts
				if sts.Spec.Template.Spec.InitContainers[0].Name != "copy-minio-client" {
					t.Errorf("MinioStatefulset() got = %v, want %v", sts.Spec.Template.Spec.InitContainers[0].Name, "copy-minio-client")
				}
				if len(sts.Spec.Template.Spec.InitContainers[0].VolumeMounts) != tt.wantNumInitContainerVolumeMounts {
					t.Errorf("MinioStatefulset() got = %v, want %v", len(sts.Spec.Template.Spec.InitContainers[0].VolumeMounts), tt.wantNumInitContainerVolumeMounts)
					return
				}

				if sts.Spec.Template.Spec.InitContainers[1].Name != "export-minio-data" {
					t.Errorf("MinioStatefulset() got = %v, want %v", sts.Spec.Template.Spec.InitContainers[1].Name, "export-minio-data")
				}
				if len(sts.Spec.Template.Spec.InitContainers[1].VolumeMounts) != tt.wantNumInitContainerVolumeMounts {
					t.Errorf("MinioStatefulset() got = %v, want %v", len(sts.Spec.Template.Spec.InitContainers[1].VolumeMounts), tt.wantNumInitContainerVolumeMounts)
					return
				}
				// export container should be using the old image
				if sts.Spec.Template.Spec.InitContainers[1].Image != tt.deployOptions.CurrentMinioImage {
					t.Errorf("MinioStatefulset() got = %v, want %v", sts.Spec.Template.Spec.InitContainers[1].Image, tt.deployOptions.CurrentMinioImage)
				}

				if sts.Spec.Template.Spec.InitContainers[2].Name != "import-minio-data" {
					t.Errorf("MinioStatefulset() got = %v, want %v", sts.Spec.Template.Spec.InitContainers[2].Name, "import-minio-data")
				}
				if len(sts.Spec.Template.Spec.InitContainers[2].VolumeMounts) != tt.wantNumInitContainerVolumeMounts {
					t.Errorf("MinioStatefulset() got = %v, want %v", len(sts.Spec.Template.Spec.InitContainers[2].VolumeMounts), tt.wantNumInitContainerVolumeMounts)
					return
				}
			}
		})
	}
}

func Test_MinioStatefulset_ResourceRequirements(t *testing.T) {
	tests := []struct {
		name          string
		deployOptions types.DeployOptions
		size          resource.Quantity
		want          corev1.ResourceRequirements
		wantErr       bool
	}{
		{
			name:          "sets resource requests and limits for non-autopilot",
			deployOptions: types.DeployOptions{},
			size:          resource.MustParse("10Gi"),
			want: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("100m"),
					"memory": resource.MustParse("512Mi"),
				},
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("50m"),
					"memory": resource.MustParse("100Mi"),
				},
			},
			wantErr: false,
		},
		{
			name: "sets resource requests and limits for autopilot",
			deployOptions: types.DeployOptions{
				IsGKEAutopilot: true,
			},
			size: resource.MustParse("10Gi"),
			want: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("250m"),
					"memory": resource.MustParse("512Mi"),
				},
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("250m"),
					"memory": resource.MustParse("512Mi"),
				},
			},
			wantErr: false,
		},
		{
			name: "sets resource requests and limits for autopilot with migration",
			deployOptions: types.DeployOptions{
				IsGKEAutopilot:   true,
				MigrateToMinioXl: true,
			},
			size: resource.MustParse("10Gi"),
			want: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse("250m"),
					"memory": resource.MustParse("512Mi"),
				},
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("250m"),
					"memory": resource.MustParse("512Mi"),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sts, err := MinioStatefulset(tt.deployOptions, tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("MinioStatefulset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if sts.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String() != tt.want.Requests.Cpu().String() {
				t.Errorf("MinioStatefulset() got = %v, want %v", sts.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String(), tt.want.Requests.Cpu().String())
			}
			if sts.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String() != tt.want.Requests.Memory().String() {
				t.Errorf("MinioStatefulset() got = %v, want %v", sts.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String(), tt.want.Requests.Memory().String())
			}
			if sts.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String() != tt.want.Limits.Cpu().String() {
				t.Errorf("MinioStatefulset() got = %v, want %v", sts.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String(), tt.want.Limits.Cpu().String())
			}
			if sts.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String() != tt.want.Limits.Memory().String() {
				t.Errorf("MinioStatefulset() got = %v, want %v", sts.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String(), tt.want.Limits.Memory().String())
			}

			if tt.deployOptions.MigrateToMinioXl {
				for _, container := range sts.Spec.Template.Spec.InitContainers {
					if container.Resources.Requests.Cpu().String() != tt.want.Requests.Cpu().String() {
						t.Errorf("MinioStatefulset() got = %v, want %v", container.Resources.Requests.Cpu().String(), tt.want.Requests.Cpu().String())
					}
					if container.Resources.Requests.Memory().String() != tt.want.Requests.Memory().String() {
						t.Errorf("MinioStatefulset() got = %v, want %v", container.Resources.Requests.Memory().String(), tt.want.Requests.Memory().String())
					}
					if container.Resources.Limits.Cpu().String() != tt.want.Limits.Cpu().String() {
						t.Errorf("MinioStatefulset() got = %v, want %v", container.Resources.Limits.Cpu().String(), tt.want.Limits.Cpu().String())
					}
					if container.Resources.Limits.Memory().String() != tt.want.Limits.Memory().String() {
						t.Errorf("MinioStatefulset() got = %v, want %v", container.Resources.Limits.Memory().String(), tt.want.Limits.Memory().String())
					}
				}
			}
		})
	}
}

func TestNodeSelectorsInMinioStatefulset(t *testing.T) {
	tests := []struct {
		name            string
		nodeSelectors   map[string]string
		expectSelectors bool
	}{
		{
			name: "with node selectors",
			nodeSelectors: map[string]string{
				"node-role.kubernetes.io/worker": "true",
				"kubernetes.io/os":               "linux",
			},
			expectSelectors: true,
		},
		{
			name:            "without node selectors",
			nodeSelectors:   map[string]string{},
			expectSelectors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployOptions := types.DeployOptions{
				Namespace:    "default",
				NodeSelector: tt.nodeSelectors,
			}

			size := resource.MustParse("4Gi")
			statefulset, err := MinioStatefulset(deployOptions, size)
			assert.NoError(t, err)

			if tt.expectSelectors {
				assert.Equal(t, tt.nodeSelectors, statefulset.Spec.Template.Spec.NodeSelector)
			} else {
				// If no node selectors are provided, the map should be nil or empty
				if statefulset.Spec.Template.Spec.NodeSelector != nil {
					assert.Empty(t, statefulset.Spec.Template.Spec.NodeSelector)
				}
			}
		})
	}
}
