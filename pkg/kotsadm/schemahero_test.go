package kotsadm

import (
	"fmt"
	"testing"
	"time"

	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func Test_getMigrationsYAML(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
	}{
		{
			name:      "no namespace",
			namespace: "",
		},
		{
			name:      "default namespace",
			namespace: "default",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			deployOptions := types.DeployOptions{
				Namespace:        test.namespace,
				PostgresPassword: fmt.Sprintf("%d", time.Now().Unix()),
			}

			manifests, err := getMigrationsYAML(deployOptions)
			req.NoError(err)
			assert.NotNil(t, manifests)

			migrations, ok := manifests["migrations.yaml"]
			assert.Equal(t, true, ok)

			decode := scheme.Codecs.UniversalDeserializer().Decode
			obj, _, err := decode(migrations, nil, nil)
			req.NoError(err)

			pod := obj.(*corev1.Pod)

			assert.Equal(t, test.namespace, pod.Namespace)
			assert.Equal(t, corev1.RestartPolicyOnFailure, pod.Spec.RestartPolicy)

			assert.Len(t, pod.Spec.Containers, 1)

			// container := pod.Spec.Containers[0]
			// postgresURI := ""
			// for _, env := range container.Env {
			// 	if env.Name == "SCHEMAHERO_URI" {
			// 		postgresURI = env.Value
			// 	}
			// }
			// assert.NotEmpty(t, postgresURI)
			// assert.Equal(t, postgresURI, fmt.Sprintf("postgresql://kotsadm:%s@kotsadm-postgres/kotsadm?connect_timeout=10&sslmode=disable", deployOptions.PostgresPassword))
		})
	}
}
