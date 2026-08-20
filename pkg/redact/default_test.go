package redact

import (
	"bytes"
	"context"
	"testing"

	"github.com/replicatedhq/kots/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func Test_CreateRenderedDefaultRedactSpec(t *testing.T) {
	defaultRedactor := getDefaultRedactor()

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var b bytes.Buffer
	if err := s.Encode(defaultRedactor, &b); err != nil {
		t.Errorf("failed to marshal default redactor: %v", err)
	}
	defaultRedactorSpec := b.String()

	tests := []struct {
		name   string
		client kubernetes.Interface
	}{
		{
			name:   "no existing secret",
			client: fake.NewSimpleClientset(),
		},
		{
			name: "existing secret with no data",
			client: fake.NewSimpleClientset(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultRedactSpecName,
					Namespace: util.PodNamespace,
				},
			}),
		},
		{
			name: "existing secret with default data key",
			client: fake.NewSimpleClientset(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultRedactSpecName,
					Namespace: util.PodNamespace,
				},
				Data: map[string][]byte{
					defaultRedactSpecDataKey: []byte(defaultRedactorSpec),
				},
			}),
		},
		{
			name: "existing legacy configmap is deleted",
			client: fake.NewSimpleClientset(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultRedactSpecName,
					Namespace: util.PodNamespace,
				},
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := CreateRenderedDefaultRedactSpec(test.client)
			require.NoError(t, err)

			secret, err := test.client.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), defaultRedactSpecName, metav1.GetOptions{})
			require.NoError(t, err)

			require.NotNil(t, secret.Data)
			got, ok := secret.Data[defaultRedactSpecDataKey]
			require.True(t, ok)

			assert.Equal(t, defaultRedactorSpec, string(got))

			_, err = test.client.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), defaultRedactSpecName, metav1.GetOptions{})
			assert.True(t, kuberneteserrors.IsNotFound(err))
		})
	}
}
