package kotsadm

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadm "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"gopkg.in/go-playground/assert.v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_getHTTPProxySettings(t *testing.T) {
	tests := []struct {
		name           string
		objects        []runtime.Object
		wantHttpProxy  string
		wantHttpsProxy string
		wantNoProxy    string
		wantErr        bool
	}{
		{
			name: "found in deployment",
			objects: []runtime.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm",
						Namespace: "kotsadm",
					},
					Spec: appsv1.DeploymentSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name: "kotsadm",
										Env: kotsadm.GetProxyEnv(types.DeployOptions{
											HTTPProxyEnvValue:  "http://proxy.com",
											HTTPSProxyEnvValue: "https://proxy.com",
											NoProxyEnvValue:    "1.2.3.4",
										}),
									},
								},
							},
						},
					},
				},
			},
			wantHttpProxy:  "http://proxy.com",
			wantHttpsProxy: "https://proxy.com",
			wantNoProxy:    "1.2.3.4,kotsadm-rqlite,kotsadm-postgres,kotsadm-minio,kotsadm-api-node",
			wantErr:        false,
		},
		{
			name: "found in statefulset",
			objects: []runtime.Object{
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm",
						Namespace: "kotsadm",
					},
					Spec: appsv1.StatefulSetSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name: "kotsadm",
										Env: kotsadm.GetProxyEnv(types.DeployOptions{
											HTTPProxyEnvValue:  "http://proxy.com",
											HTTPSProxyEnvValue: "https://proxy.com",
											NoProxyEnvValue:    "1.2.3.4",
										}),
									},
								},
							},
						},
					},
				},
			},
			wantHttpProxy:  "http://proxy.com",
			wantHttpsProxy: "https://proxy.com",
			wantNoProxy:    "1.2.3.4,kotsadm-rqlite,kotsadm-postgres,kotsadm-minio,kotsadm-api-node",
			wantErr:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := fake.NewSimpleClientset(tt.objects...)

			gotHttpProxy, gotHttpsProxy, gotNoProxy, err := getHTTPProxySettings("kotsadm", cli)
			if (err != nil) != tt.wantErr {
				t.Errorf("getHTTPProxySettings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, gotHttpProxy, tt.wantHttpProxy)
			assert.Equal(t, gotHttpsProxy, tt.wantHttpsProxy)
			assert.Equal(t, gotNoProxy, tt.wantNoProxy)
		})
	}
}

func Test_hasStrictSecurityContext(t *testing.T) {
	tests := []struct {
		name    string
		objects []runtime.Object
		want    bool
		wantErr bool
	}{
		{
			name: "strict",
			objects: []runtime.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm",
						Namespace: "kotsadm",
					},
					Spec: appsv1.DeploymentSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name: "kotsadm",
									},
								},
								SecurityContext: k8sutil.SecurePodContext(1001, 1001, true),
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "not strict",
			objects: []runtime.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm",
						Namespace: "kotsadm",
					},
					Spec: appsv1.DeploymentSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name: "kotsadm",
									},
								},
								SecurityContext: k8sutil.SecurePodContext(1001, 1001, false),
							},
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "nil",
			objects: []runtime.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm",
						Namespace: "kotsadm",
					},
					Spec: appsv1.DeploymentSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
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
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := fake.NewSimpleClientset(tt.objects...)

			got, err := hasStrictSecurityContext("kotsadm", cli)
			if (err != nil) != tt.wantErr {
				t.Errorf("hasStrictSecurityContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, got, tt.want)
		})
	}
}
