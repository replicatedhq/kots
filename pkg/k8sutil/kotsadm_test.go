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
			name: "configmap exists - returns configmap value",
			args: args{
				clientset: fake.NewClientset(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: KotsadmIDConfigMapName},
					Data:       map[string]string{"id": "cluster-id-from-configmap"},
				}),
			},
			want:                  "cluster-id-from-configmap",
			shouldCreateConfigMap: false,
		},
		{
			name: "configmap does not exist, no store initialized - generates new id",
			args: args{
				clientset: fake.NewClientset(),
			},
			want:                  "",
			shouldCreateConfigMap: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure no cluster ID provider is registered for these tests
			SetClusterIDProvider(nil)
			defer SetClusterIDProvider(nil)

			got := GetKotsadmID(tt.args.clientset)
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
				// ConfigMap should contain the returned value (the generated ID)
				assert.Equal(t, got, cm.Data["id"])
			}
		})
	}
}

func TestGetKotsadmID_WithStoreProvider(t *testing.T) {
	// Test that when store is available, it's used as a fallback
	clientset := fake.NewClientset() // No ConfigMap

	// Register a mock store provider
	mockClusterID := "cluster-id-from-store"
	SetClusterIDProvider(func() string {
		return mockClusterID
	})
	defer SetClusterIDProvider(nil)

	got := GetKotsadmID(clientset)
	assert.Equal(t, mockClusterID, got)

	// Should have created ConfigMap with the store's cluster ID
	cm, err := clientset.CoreV1().ConfigMaps("").Get(context.TODO(), KotsadmIDConfigMapName, metav1.GetOptions{})
	assert.Equal(t, nil, err)
	assert.Equal(t, mockClusterID, cm.Data["id"])
}
