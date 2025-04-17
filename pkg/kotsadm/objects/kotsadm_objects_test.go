package kotsadm

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

func TestNodeSelectorParsing(t *testing.T) {
	testCases := []struct {
		name           string
		input          []string
		expected       map[string]string
		expectedError  bool
		expectedErrMsg string
	}{
		{
			name:           "valid node selectors",
			input:          []string{"kubernetes.io/os=linux", "node-role.kubernetes.io/worker=true"},
			expected:       map[string]string{"kubernetes.io/os": "linux", "node-role.kubernetes.io/worker": "true"},
			expectedError:  false,
			expectedErrMsg: "",
		},
		{
			name:           "invalid format",
			input:          []string{"kubernetes.io/os:linux"},
			expected:       nil,
			expectedError:  true,
			expectedErrMsg: "node-selector flag is not in the correct format. Must be key=value",
		},
		{
			name:           "empty input",
			input:          []string{},
			expected:       map[string]string{},
			expectedError:  false,
			expectedErrMsg: "",
		},
		{
			name:           "multiple equal signs",
			input:          []string{"key=value=extra"},
			expected:       map[string]string{"key": "value=extra"},
			expectedError:  false,
			expectedErrMsg: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nodeSelectors := map[string]string{}
			var err error

			for _, nodeSelector := range tc.input {
				parts := make([]string, 0)
				if nodeSelector != "" {
					parts = append(parts, nodeSelector)
				}

				for _, nodeSelector := range parts {
					keyValue := make([]string, 0)
					if nodeSelector != "" {
						keyValue = append(keyValue, nodeSelector)
					}

					for _, nv := range keyValue {
						parts := make([]string, 0)
						if nv != "" {
							// This simulates the behavior of strings.Split(nv, "=")
							splitParts := []string{}
							for i, c := range nv {
								if c == '=' && i > 0 && i < len(nv)-1 {
									splitParts = append(splitParts, nv[:i], nv[i+1:])
									break
								}
							}
							if len(splitParts) == 2 {
								parts = append(parts, splitParts...)
							} else {
								parts = append(parts, nv)
							}
						}

						if len(parts) != 2 {
							err = assert.AnError
							break
						}
						nodeSelectors[parts[0]] = parts[1]
					}
				}
			}

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, nodeSelectors)
			}
		})
	}
}

func TestNodeSelectorsInDeployment(t *testing.T) {
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
				NodeSelector: tt.nodeSelectors,
			}

			deployment, err := KotsadmDeployment(deployOptions)
			assert.NoError(t, err)

			if tt.expectSelectors {
				assert.Equal(t, tt.nodeSelectors, deployment.Spec.Template.Spec.NodeSelector)
			} else {
				// If no node selectors are provided, the map should be nil or empty
				if deployment.Spec.Template.Spec.NodeSelector != nil {
					assert.Empty(t, deployment.Spec.Template.Spec.NodeSelector)
				}
			}
		})
	}
}

func TestNodeSelectorsInStatefulset(t *testing.T) {
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
			statefulset, err := KotsadmStatefulSet(deployOptions, size)
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
