package kotsadm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_updateKotsadmStatefulSetScriptsPath(t *testing.T) {
	type args struct {
		existing *appsv1.StatefulSet
	}
	tests := []struct {
		name string
		args args
		want *appsv1.StatefulSet
	}{
		{
			name: "migrate scripts dir",
			args: args{
				existing: &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kotsadm",
					},
					Spec: appsv1.StatefulSetSpec{
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{
									"backup.velero.io/backup-volumes":   "backup",
									"pre.hook.backup.velero.io/command": `["/backup.sh"]`,
									"pre.hook.backup.velero.io/timeout": "10m",
								},
							},
							Spec: corev1.PodSpec{
								InitContainers: []corev1.Container{
									{
										Name: "some-other-init-container",
									},
									{
										Name: "restore-data",
										Command: []string{
											"/restore.sh",
										},
									},
									{
										Name: "migrate-s3",
										Command: []string{
											"/migrate-s3.sh",
										},
									},
								},
								Containers: []corev1.Container{
									{
										Name: "kotsadm",
									},
								},
							},
						},
					},
				},
			},
			want: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kotsadm",
				},
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"backup.velero.io/backup-volumes":   "backup",
								"pre.hook.backup.velero.io/command": `["/scripts/backup.sh"]`,
								"pre.hook.backup.velero.io/timeout": "10m",
							},
						},
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{
									Name: "some-other-init-container",
								},
								{
									Name: "restore-data",
									Command: []string{
										"/scripts/restore.sh",
									},
								},
								{
									Name: "migrate-s3",
									Command: []string{
										"/scripts/migrate-s3.sh",
									},
								},
							},
							Containers: []corev1.Container{
								{
									Name: "kotsadm",
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateKotsadmStatefulSetScriptsPath(tt.args.existing)
			assert.Equal(t, tt.want, tt.args.existing)
		})
	}
}

func Test_updateKotsadmDeploymentScriptsPath(t *testing.T) {
	type args struct {
		existing *appsv1.Deployment
	}
	tests := []struct {
		name string
		args args
		want *appsv1.Deployment
	}{
		{
			name: "migrate scripts dir",
			args: args{
				existing: &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kotsadm",
					},
					Spec: appsv1.DeploymentSpec{
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{
									"backup.velero.io/backup-volumes":   "backup",
									"pre.hook.backup.velero.io/command": `["/backup.sh"]`,
									"pre.hook.backup.velero.io/timeout": "10m",
								},
							},
							Spec: corev1.PodSpec{
								InitContainers: []corev1.Container{
									{
										Name: "some-other-init-container",
									},
									{
										Name: "restore-db",
										Command: []string{
											"/restore-db.sh",
										},
									},
									{
										Name: "restore-s3",
										Command: []string{
											"/restore-s3.sh",
										},
									},
								},
								Containers: []corev1.Container{
									{
										Name: "kotsadm",
										Env: []corev1.EnvVar{
											{
												Name:  "POSTGRES_SCHEMA_DIR",
												Value: "/postgres/tables",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kotsadm",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"backup.velero.io/backup-volumes":   "backup",
								"pre.hook.backup.velero.io/command": `["/scripts/backup.sh"]`,
								"pre.hook.backup.velero.io/timeout": "10m",
							},
						},
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{
									Name: "some-other-init-container",
								},
								{
									Name: "restore-db",
									Command: []string{
										"/scripts/restore-db.sh",
									},
								},
								{
									Name: "restore-s3",
									Command: []string{
										"/scripts/restore-s3.sh",
									},
								},
							},
							Containers: []corev1.Container{
								{
									Name: "kotsadm",
									Env: []corev1.EnvVar{
										{
											Name:  "POSTGRES_SCHEMA_DIR",
											Value: "/scripts/postgres/tables",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateKotsadmDeploymentScriptsPath(tt.args.existing)
			assert.Equal(t, tt.want, tt.args.existing)
		})
	}
}
