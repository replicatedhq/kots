package kotsutil

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestHasEmbeddedRegistry(t *testing.T) {
	type args struct {
		clientset kubernetes.Interface
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "default namespace with correct type",
			args: args{
				clientset: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "registry-creds",
						Namespace: "default",
					},
					Type: corev1.SecretTypeDockerConfigJson,
				}),
			},
			want: true,
		},
		{
			name: "kotsadm namespace with correct type",
			args: args{
				clientset: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "registry-creds",
						Namespace: "kotsadm",
					},
					Type: corev1.SecretTypeDockerConfigJson,
				}),
			},
			want: true,
		},
		{
			name: "default namespace with incorrect type",
			args: args{
				clientset: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "registry-creds",
						Namespace: "default",
					},
					Type: corev1.SecretTypeOpaque,
				}),
			},
			want: false,
		},
		{
			name: "kotsadm namespace with incorrect type",
			args: args{
				clientset: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "registry-creds",
						Namespace: "kotsadm",
					},
					Type: corev1.SecretTypeOpaque,
				}),
			},
			want: false,
		},
		{
			name: "incorrect namespace but correct type",
			args: args{
				clientset: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "registry-creds",
						Namespace: "incorrect",
					},
					Type: corev1.SecretTypeDockerConfigJson,
				}),
			},
			want: false,
		},
		{
			name: "incorrect namespace and type",
			args: args{
				clientset: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "registry-creds",
						Namespace: "incorrect",
					},
					Type: corev1.SecretTypeOpaque,
				}),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasEmbeddedRegistry(tt.args.clientset); got != tt.want {
				t.Errorf("HasEmbeddedRegistry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEmbeddedRegistryCreds(t *testing.T) {
	type args struct {
		clientset kubernetes.Interface
	}
	tests := []struct {
		name     string
		args     args
		wantHost string
		wantUser string
		wantPass string
	}{
		{
			name: "default namespace with correct type and user",
			args: args{
				clientset: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "registry-creds",
						Namespace: "default",
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						".dockerconfigjson": []byte(`{"auths":{"host":{"username":"kurl","password":"password"}}}`),
					},
				}),
			},
			wantHost: "host",
			wantUser: "kurl",
			wantPass: "password",
		},
		{
			name: "default namespace with correct type but incorrect user",
			args: args{
				clientset: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "registry-creds",
						Namespace: "default",
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						".dockerconfigjson": []byte(`{"auths":{"host":{"username":"incorrect","password":"password"}}}`),
					},
				}),
			},
			wantHost: "",
			wantUser: "",
			wantPass: "",
		},
		{
			name: "kotsadm namespace with correct type and user",
			args: args{
				clientset: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "registry-creds",
						Namespace: "kotsadm",
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						".dockerconfigjson": []byte(`{"auths":{"host":{"username":"embedded-cluster","password":"password"}}}`),
					},
				}),
			},
			wantHost: "host",
			wantUser: "embedded-cluster",
			wantPass: "password",
		},
		{
			name: "kotsadm namespace with correct type but incorrect user",
			args: args{
				clientset: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "registry-creds",
						Namespace: "kotsadm",
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						".dockerconfigjson": []byte(`{"auths":{"host":{"username":"incorrect","password":"password"}}}`),
					},
				}),
			},
			wantHost: "",
			wantUser: "",
			wantPass: "",
		},
		{
			name: "default namespace with incorrect type",
			args: args{
				clientset: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "registry-creds",
						Namespace: "default",
					},
					Type: corev1.SecretTypeOpaque,
				}),
			},
			wantHost: "",
			wantUser: "",
			wantPass: "",
		},
		{
			name: "kotsadm namespace with incorrect type",
			args: args{
				clientset: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "registry-creds",
						Namespace: "kotsadm",
					},
					Type: corev1.SecretTypeOpaque,
				}),
			},
			wantHost: "",
			wantUser: "",
			wantPass: "",
		},
		{
			name: "incorrect namespace with correct type and user",
			args: args{
				clientset: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "registry-creds",
						Namespace: "incorrect",
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						".dockerconfigjson": []byte(`{"auths":{"host":{"username":"kurl","password":"password"}}}`),
					},
				}),
			},
			wantHost: "",
			wantUser: "",
			wantPass: "",
		},
		{
			name: "incorrect namespace with correct type but incorrect user",
			args: args{
				clientset: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "registry-creds",
						Namespace: "incorrect",
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						".dockerconfigjson": []byte(`{"auths":{"host":{"username":"incorrect","password":"password"}}}`),
					},
				}),
			},
			wantHost: "",
			wantUser: "",
			wantPass: "",
		},
		{
			name: "incorrect namespace and type",
			args: args{
				clientset: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "registry-creds",
						Namespace: "incorrect",
					},
					Type: corev1.SecretTypeOpaque,
				}),
			},
			wantHost: "",
			wantUser: "",
			wantPass: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHost, gotUser, gotPass := GetEmbeddedRegistryCreds(tt.args.clientset)
			if gotHost != tt.wantHost {
				t.Errorf("GetEmbeddedRegistryCreds() gotHost = %v, want %v", gotHost, tt.wantHost)
			}
			if gotUser != tt.wantUser {
				t.Errorf("GetEmbeddedRegistryCreds() gotUser = %v, want %v", gotUser, tt.wantUser)
			}
			if gotPass != tt.wantPass {
				t.Errorf("GetEmbeddedRegistryCreds() gotPass = %v, want %v", gotPass, tt.wantPass)
			}
		})
	}
}
