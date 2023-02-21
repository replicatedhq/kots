package kotsadm

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func Test_getRqliteYAML(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		password  string
	}{
		{
			name:      "no namespace",
			namespace: "",
			password:  "test",
		},
		{
			name:      "default namespace",
			namespace: "default",
			password:  "test",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			manifests, err := getRqliteYAML(types.DeployOptions{
				Namespace:      test.namespace,
				RqlitePassword: test.password,
			})
			req.NoError(err)
			assert.NotNil(t, manifests)

			decode := scheme.Codecs.UniversalDeserializer().Decode

			statefulSetManifest, ok := manifests["rqlite-statefulset.yaml"]
			assert.Equal(t, true, ok)
			obj, _, err := decode(statefulSetManifest, nil, nil)
			req.NoError(err)
			statefulSet := obj.(*appsv1.StatefulSet)

			serviceManifest, ok := manifests["rqlite-service.yaml"]
			assert.Equal(t, true, ok)
			obj, _, err = decode(serviceManifest, nil, nil)
			req.NoError(err)
			service := obj.(*corev1.Service)

			headlessServiceManifest, ok := manifests["rqlite-headless-service.yaml"]
			assert.Equal(t, true, ok)
			obj, _, err = decode(headlessServiceManifest, nil, nil)
			req.NoError(err)
			headlessService := obj.(*corev1.Service)

			assert.Len(t, statefulSet.Spec.VolumeClaimTemplates, 1)

			assert.Equal(t, service.Spec.Type, corev1.ServiceTypeClusterIP)

			assert.Equal(t, headlessService.Spec.Type, corev1.ServiceTypeClusterIP)
			assert.Equal(t, headlessService.Spec.ClusterIP, corev1.ClusterIPNone)
		})
	}
}
