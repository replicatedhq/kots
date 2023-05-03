package kurl

import (
	"testing"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
)

func mockClientWithError(err error) kubernetes.Interface {
	mockClient := fake.Clientset{}
	mockClient.AddReactor("get", "configmaps", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, err
	})
	return &mockClient
}

func Test_IsKurl(t *testing.T) {
	tests := []struct {
		name    string
		args    kubernetes.Interface
		want    bool
		wantErr bool
	}{
		{
			name:    "expect error when client is nil",
			args:    nil,
			want:    false,
			wantErr: true,
		},
		{
			name: "expect false when configmap is not found",
			args: mockClientWithError(kuberneteserrors.NewNotFound(corev1.Resource("configmaps"), configMapName)),
			want: false,
		},
		{
			name: "expect false when client is unauthorized",
			args: mockClientWithError(kuberneteserrors.NewUnauthorized("Unauthorized")),
			want: false,
		},
		{
			name: "expect false when client is forbidden",
			args: mockClientWithError(kuberneteserrors.NewForbidden(corev1.Resource("configmaps"), configMapName, errors.New("Forbidden"))),
			want: false,
		},
		{
			name:    "expect error when client returns internal error",
			args:    mockClientWithError(kuberneteserrors.NewInternalError(errors.New("Internal Error"))),
			want:    false,
			wantErr: true,
		},
		{
			name: "expect true when configmap is found",
			args: fake.NewSimpleClientset(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configMapName,
					Namespace: metav1.NamespaceSystem,
				},
			}),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsKurl(tt.args)
			if err != nil && !tt.wantErr {
				t.Errorf("IsKurl() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("IsKurl() = %v, want %v", got, tt.want)
			}
		})
	}
}
