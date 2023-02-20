package redact

import (
	"bytes"
	"context"
	"testing"

	"github.com/replicatedhq/kots/pkg/util"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
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
		want   string
		client kubernetes.Interface
	}{
		{
			name:   "no existing default configmap",
			want:   defaultRedactorSpec,
			client: fake.NewSimpleClientset(),
		},
		{
			name: "existing default configmap with no data",
			want: defaultRedactorSpec,
			client: fake.NewSimpleClientset(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultRedactSpecConfigMapName,
					Namespace: util.PodNamespace,
				},
			}),
		},
		{
			name: "existing default configmap with no default data key",
			want: defaultRedactorSpec,
			client: fake.NewSimpleClientset(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultRedactSpecConfigMapName,
					Namespace: util.PodNamespace,
				},
				Data: map[string]string{},
			}),
		},
		{
			name: "existing default configmap with default data key",
			want: defaultRedactorSpec,
			client: fake.NewSimpleClientset(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultRedactSpecConfigMapName,
					Namespace: util.PodNamespace,
				},
				Data: map[string]string{
					defaultRedactSpecDataKey: defaultRedactorSpec,
				},
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := CreateRenderedDefaultRedactSpec(test.client)
			assert.NoErrorf(t, err, "failed to create default redactor configmap: %v", err)

			configMap, err := test.client.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), defaultRedactSpecConfigMapName, metav1.GetOptions{})
			assert.NoErrorf(t, err, "failed to get default redactor configmap: %v", err)

			if configMap.Data == nil {
				t.Errorf("expected data to be set")
			}

			got, ok := configMap.Data[defaultRedactSpecDataKey]
			if !ok {
				t.Errorf("no default redactor data key")
			}

			assert.Equalf(t, test.want, got, "expected default redactor spec to be %s, got %s", test.want, got)
		})
	}
}
