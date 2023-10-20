package k8sutil

import (
	"context"
	"testing"

	"gopkg.in/go-playground/assert.v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetKotsadmID(t *testing.T) {

	type args struct {
		clientset kubernetes.Interface
	}
	tests := []struct {
		name                  string
		args                  args
		want                  string
		shouldCreateConfigMap bool
	}{
		{
			name: "configmap exists",
			args: args{
				clientset: fake.NewSimpleClientset(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: KotsadmIDConfigMapName},
					Data:       map[string]string{"id": "cluster-id"},
				}),
			},
			want:                  "cluster-id",
			shouldCreateConfigMap: false,
		},
		{
			name: "configmap does not exist, should create",
			args: args{
				clientset: fake.NewSimpleClientset(),
			},
			want:                  "",
			shouldCreateConfigMap: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetKotsadmID(tt.args.clientset)
			if tt.want != "" {
				assert.Equal(t, tt.want, got)
			} else {
				// a random uuid is generated
				assert.NotEqual(t, "", got)
			}

			if tt.shouldCreateConfigMap {
				// should have created the configmap if it didn't exist
				_, err := tt.args.clientset.CoreV1().ConfigMaps("").Get(context.TODO(), KotsadmIDConfigMapName, metav1.GetOptions{})
				assert.Equal(t, nil, err)
			}
		})
	}
}

func Test_GetKotsadmDeploymentUID(t *testing.T) {
	type args struct {
		clientset kubernetes.Interface
		namespace string
	}
	tests := []struct {
		name    string
		args    args
		want    apimachinerytypes.UID
		wantErr bool
	}{
		{
			name: "deployment exists",
			args: args{
				clientset: fake.NewSimpleClientset(&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kotsadm",
						Namespace: "default",
						UID:       "test-uid",
					},
				}),
				namespace: "default",
			},
			want: "test-uid",
		},
		{
			name: "deployment does not exist",
			args: args{
				clientset: fake.NewSimpleClientset(),
				namespace: "default",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetKotsadmDeploymentUID(tt.args.clientset, tt.args.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetKotsadmDeploymentUID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
