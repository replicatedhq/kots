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
		clientset         kubernetes.Interface
		databaseClusterID string
	}
	tests := []struct {
		name                  string
		args                  args
		want                  string
		shouldCreateConfigMap bool
	}{
		{
			name: "configmap exists - returns configmap value",
			args: args{
				clientset: fake.NewClientset(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: KotsadmIDConfigMapName},
					Data:       map[string]string{"id": "cluster-id-from-configmap"},
				}),
				databaseClusterID: "cluster-id-from-database",
			},
			want:                  "cluster-id-from-configmap",
			shouldCreateConfigMap: false,
		},
		{
			name: "configmap does not exist, database cluster id provided - uses database value",
			args: args{
				clientset:         fake.NewClientset(),
				databaseClusterID: "cluster-id-from-database",
			},
			want:                  "cluster-id-from-database",
			shouldCreateConfigMap: true,
		},
		{
			name: "configmap does not exist, no database cluster id - generates new id",
			args: args{
				clientset:         fake.NewClientset(),
				databaseClusterID: "",
			},
			want:                  "",
			shouldCreateConfigMap: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetKotsadmID(tt.args.clientset, tt.args.databaseClusterID)
			if tt.want != "" {
				assert.Equal(t, tt.want, got)
			} else {
				// a random uuid is generated
				assert.NotEqual(t, "", got)
			}

			if tt.shouldCreateConfigMap {
				// should have created the configmap if it didn't exist
				cm, err := tt.args.clientset.CoreV1().ConfigMaps("").Get(context.TODO(), KotsadmIDConfigMapName, metav1.GetOptions{})
				assert.Equal(t, nil, err)
				// ConfigMap should contain the returned value
				if tt.want != "" {
					assert.Equal(t, tt.want, cm.Data["id"])
				}
			}
		})
	}
}
