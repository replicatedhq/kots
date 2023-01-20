package k8sutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_addLabelsToSecret(t *testing.T) {
	tests := []struct {
		name             string
		secret           []runtime.Object
		targetNamespace  string
		targetSecretName string
		additionalLabels map[string]string
		expectedLabels   map[string]string
		expectSuccess    bool
	}{
		{
			name: "didn't add duplicate labels with matching secret name",
			secret: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-1",
						Namespace: "default",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
					Data: map[string][]byte{
						"spec": []byte("spec-1"),
					},
				},
			},
			additionalLabels: map[string]string{
				"foo": "bar",
			},
			targetNamespace:  "default",
			targetSecretName: "secret-1",
			expectedLabels: map[string]string{
				"foo": "bar",
			},
			expectSuccess: true,
		},
		{
			name: "add additional labels with matching secret name",
			secret: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-1",
						Namespace: "default",
						Labels: map[string]string{
							"foo": "foo",
						},
					},
					Data: map[string][]byte{
						"spec": []byte("spec-1"),
					},
				},
			},
			additionalLabels: map[string]string{
				"bar": "bar",
			},
			targetNamespace:  "default",
			targetSecretName: "secret-1",
			expectedLabels: map[string]string{
				"foo": "foo",
				"bar": "bar",
			},
			expectSuccess: true,
		},
		{
			name: "add additional troubleshoot labels with matching secret name",
			secret: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-1",
						Namespace: "default",
						Labels: map[string]string{
							"kots.io/backup":  "velero",
							"kots.io/kotsadm": "true",
						},
					},
					Data: map[string][]byte{
						"spec": []byte("spec-1"),
					},
				},
			},
			additionalLabels: map[string]string{
				"troubleshoot.io/kind": "support-bundle",
			},
			targetNamespace:  "default",
			targetSecretName: "secret-1",
			expectedLabels: map[string]string{
				"kots.io/backup":       "velero",
				"kots.io/kotsadm":      "true",
				"troubleshoot.io/kind": "support-bundle",
			},
			expectSuccess: true,
		},
		{
			name: "return nil with not matching secret name",
			secret: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-1",
						Namespace: "default",
						Labels: map[string]string{
							"foo": "foo",
						},
					},
					Data: map[string][]byte{
						"spec": []byte("spec-1"),
					},
				},
			},
			additionalLabels: map[string]string{
				"bar": "bar",
			},
			targetNamespace:  "default",
			targetSecretName: "secret-2",
			expectSuccess:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClientset := fake.NewSimpleClientset(tt.secret...)
			targetSecret, err := AddLabelsToSecret(fakeClientset, tt.targetNamespace, tt.targetSecretName, tt.additionalLabels)
			if err != nil && tt.expectSuccess {
				t.Errorf("AddLabelExistingSpecSecret() error = %v, expectSuccess %v", err, tt.expectSuccess)
			} else if err == nil && !tt.expectSuccess {
				t.Errorf("AddLabelExistingSpecSecret() error = nil, expectSuccess %v", tt.expectSuccess)
			} else if targetSecret != nil {
				assert.Equal(t, tt.expectedLabels, targetSecret.Labels)
			}
		})
	}
}
