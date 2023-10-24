package k8sutil

import (
	"context"
	"testing"

	"gopkg.in/go-playground/assert.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
