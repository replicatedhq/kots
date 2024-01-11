package kotsadm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func Test_cleanUpMigrationArtifact(t *testing.T) {
	tests := []struct {
		name        string
		namespace   string
		hasArtifact bool
	}{
		{
			name:        "has artifact",
			namespace:   "default",
			hasArtifact: true,
		},
		{
			name:        "no artifact",
			namespace:   "default",
			hasArtifact: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			clientset := testclient.NewSimpleClientset()

			if test.hasArtifact {
				err := createMigrationArtifact(clientset, test.namespace)
				req.NoError(err)
			}

			err := cleanUpMigrationArtifact(clientset, test.namespace)
			req.NoError(err)
		})
	}
}

func Test_IsMinioXlMigrationNeeded(t *testing.T) {
	minioStsWithImage := func(image string) *appsv1.StatefulSet {
		return &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kotsadm-minio",
				Namespace: "default",
			},
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "kotsadm-minio",
								Image: image,
							},
						},
					},
				},
			},
		}
	}

	tests := []struct {
		name           string
		clientset      kubernetes.Interface
		wantMigration  bool
		wantMinioImage string
		wantErr        bool
	}{
		{
			name:           "should migrate old image",
			clientset:      fake.NewSimpleClientset(minioStsWithImage("minio/minio:RELEASE.2020-10-27T00-54-19Z")),
			wantMigration:  true,
			wantMinioImage: "minio/minio:RELEASE.2020-10-27T00-54-19Z",
			wantErr:        false,
		},
		{
			name:           "should not migrate newer image",
			clientset:      fake.NewSimpleClientset(minioStsWithImage("minio/minio:RELEASE.2023-02-10T18-48-39Z")),
			wantMigration:  false,
			wantMinioImage: "minio/minio:RELEASE.2023-02-10T18-48-39Z",
			wantErr:        false,
		},
		{
			name:           "should not migrate image built with chainguard",
			clientset:      fake.NewSimpleClientset(minioStsWithImage("kotsadm/minio:0.20231101.183725")),
			wantMigration:  false,
			wantMinioImage: "kotsadm/minio:0.20231101.183725",
			wantErr:        false,
		},
		{
			name:           "should not migrate if no minio",
			clientset:      fake.NewSimpleClientset(),
			wantMigration:  false,
			wantMinioImage: "",
			wantErr:        false,
		},
		{
			name:           "should error if minio tag is invalid",
			clientset:      fake.NewSimpleClientset(minioStsWithImage("minio/minio:invalid")),
			wantMigration:  false,
			wantMinioImage: "",
			wantErr:        true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			needsMigration, minioImage, err := IsMinioXlMigrationNeeded(test.clientset, "default")
			if test.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}
			req.Equal(test.wantMigration, needsMigration)
			req.Equal(test.wantMinioImage, minioImage)
		})
	}
}

func Test_IsMinioXlMigrationRunning(t *testing.T) {
	tests := []struct {
		name      string
		clientset kubernetes.Interface
		want      bool
	}{
		{
			name: "should return true if status is running",
			clientset: fake.NewSimpleClientset(
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      MinioXlMigrationStatusConfigmapName,
						Namespace: "default",
					},
					Data: map[string]string{
						"status": "running",
					},
				},
			),
			want: true,
		},
		{
			name: "should return false if status is not running",
			clientset: fake.NewSimpleClientset(
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      MinioXlMigrationStatusConfigmapName,
						Namespace: "default",
					},
					Data: map[string]string{
						"status": "not-running",
					},
				},
			),
			want: false,
		},
		{
			name: "should return false if there is no configmap data",
			clientset: fake.NewSimpleClientset(
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      MinioXlMigrationStatusConfigmapName,
						Namespace: "default",
					},
				},
			),
			want: false,
		},
		{
			name:      "should return false if there is no status configmap",
			clientset: fake.NewSimpleClientset(),
			want:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			ctx := context.Background()
			got, err := IsMinioXlMigrationRunning(ctx, test.clientset, "default")
			req.NoError(err)
			req.Equal(test.want, got)
		})
	}
}

func Test_MarkMinioXlMigrationComplete(t *testing.T) {
	tests := []struct {
		name         string
		clientset    kubernetes.Interface
		wantComplete bool
	}{
		{
			name: "should update status to complete if running",
			clientset: fake.NewSimpleClientset(
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      MinioXlMigrationStatusConfigmapName,
						Namespace: "default",
					},
					Data: map[string]string{
						"status": "running",
					},
				},
			),
			wantComplete: true,
		},
		{
			name:         "should no-op if there is no status configmap",
			clientset:    fake.NewSimpleClientset(),
			wantComplete: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			ctx := context.Background()
			err := MarkMinioXlMigrationComplete(ctx, test.clientset, "default")
			req.NoError(err)

			if test.wantComplete {
				cm, err := test.clientset.CoreV1().ConfigMaps("default").Get(ctx, MinioXlMigrationStatusConfigmapName, metav1.GetOptions{})
				req.NoError(err)
				req.Equal("complete", cm.Data["status"])
			}
		})
	}
}
