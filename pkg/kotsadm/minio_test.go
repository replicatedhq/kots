package kotsadm

import (
	"context"
	"testing"
	"time"

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

func Test_waitForMinioInitContainers(t *testing.T) {
	tests := []struct {
		name                  string
		namespace             string
		clientset             kubernetes.Interface
		timeout               time.Duration
		desiredInitContainers int
		wantErr               bool
	}{
		{
			name:      "timeout if pods in kotsadm-minio statefulset do not have init containers",
			namespace: "default",
			clientset: fake.NewSimpleClientset(
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm-minio",
						Namespace: "default",
					},
					Spec: appsv1.StatefulSetSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "kotsadm-minio",
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm-minio-0",
						Namespace: "default",
						Labels: map[string]string{
							"app": "kotsadm-minio",
						},
					},
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{},
					},
				},
			),
			timeout:               time.Second,
			desiredInitContainers: 1,
			wantErr:               true,
		},
		{
			name:      "return if pods in kotsadm-minio statefulset have init containers",
			namespace: "default",
			clientset: fake.NewSimpleClientset(
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm-minio",
						Namespace: "default",
					},
					Spec: appsv1.StatefulSetSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "kotsadm-minio",
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm-minio-0",
						Namespace: "default",
						Labels: map[string]string{
							"app": "kotsadm-minio",
						},
					},
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{
								Name: "init",
							},
						},
					},
				},
			),
			timeout:               time.Second,
			desiredInitContainers: 1,
			wantErr:               false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			ctx := context.Background()

			err := waitForMinioInitContainers(ctx, test.namespace, test.clientset, test.timeout, test.desiredInitContainers)
			if test.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}
		})
	}
}
