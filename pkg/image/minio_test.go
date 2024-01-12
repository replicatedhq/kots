package image

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_GetMinioImage(t *testing.T) {
	tests := []struct {
		name             string
		clientset        kubernetes.Interface
		kotsadmNamespace string
		wantImage        string
		wantErr          bool
	}{
		{
			name:             "should return static image for non-kurl instance",
			clientset:        fake.NewSimpleClientset(),
			kotsadmNamespace: metav1.NamespaceDefault,
			wantImage:        Minio,
			wantErr:          false,
		},
		{
			name: "should return static image for kurl instance with non-default namespace",
			clientset: fake.NewSimpleClientset(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kurl.ConfigMapName,
					Namespace: kurl.ConfigMapNamespace,
				},
			}),
			kotsadmNamespace: "custom-namespace",
			wantImage:        Minio,
			wantErr:          false,
		},
		{
			name: "should return minio image from deployment for kurl instance with default namespace",
			clientset: fake.NewSimpleClientset(
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      kurl.ConfigMapName,
						Namespace: kurl.ConfigMapNamespace,
					},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "minio",
						Namespace: "minio",
					},
					Spec: appsv1.DeploymentSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "minio",
										Image: "minio/minio:RELEASE.2020-10-27T00-54-19Z",
									},
								},
							},
						},
					},
				},
			),
			kotsadmNamespace: metav1.NamespaceDefault,
			wantImage:        "minio/minio:RELEASE.2020-10-27T00-54-19Z",
			wantErr:          false,
		},
		{
			name: "should return minio image from statefulset for kurl instance with default namespace",
			clientset: fake.NewSimpleClientset(
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      kurl.ConfigMapName,
						Namespace: kurl.ConfigMapNamespace,
					},
				},
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ha-minio",
						Namespace: "minio",
					},
					Spec: appsv1.StatefulSetSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "minio",
										Image: "minio/minio:RELEASE.2020-10-27T00-54-19Z",
									},
								},
							},
						},
					},
				}),
			kotsadmNamespace: metav1.NamespaceDefault,
			wantImage:        "minio/minio:RELEASE.2020-10-27T00-54-19Z",
			wantErr:          false,
		},
		{
			name: "should return empty image if deployment and statefulset don't exist for kurl instance with default namespace",
			clientset: fake.NewSimpleClientset(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kurl.ConfigMapName,
					Namespace: kurl.ConfigMapNamespace,
				},
			}),
			kotsadmNamespace: metav1.NamespaceDefault,
			wantImage:        "",
			wantErr:          false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			gotImage, err := GetMinioImage(test.clientset, test.kotsadmNamespace)
			if test.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}
			req.Equal(test.wantImage, gotImage)
		})
	}
}
