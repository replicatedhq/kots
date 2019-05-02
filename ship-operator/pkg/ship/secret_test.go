package ship

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetShipWatchInstanceNamesFromMeta(t *testing.T) {
	var tests = []struct {
		name   string
		input  metav1.Object
		answer []string
	}{
		{
			name:   "no ship meta",
			answer: nil,
			input: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myappsecret",
					Namespace: metav1.NamespaceDefault,
				},
			},
		},
		{
			name:   "one shipwatch name",
			answer: []string{"myapp"},
			input: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myappsecret",
					Namespace: metav1.NamespaceDefault,
					Labels: map[string]string{
						"shipwatch": "",
					},
					Annotations: map[string]string{
						"shipwatch": "myapp",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := GetShipWatchInstanceNamesFromMeta(test.input)
			if !reflect.DeepEqual(output, test.answer) {
				t.Errorf("got %+v(%d), want %+v(%d)", output, len(output), test.answer, len(test.answer))
			}
		})
	}
}

func TestHasSecretMeta(t *testing.T) {
	var tests = []struct {
		name         string
		input        metav1.Object
		instanceName string
		answer       bool
	}{
		{
			name:         "exact",
			answer:       true,
			instanceName: "myapp",
			input: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myappsecret",
					Namespace: metav1.NamespaceDefault,
					Labels: map[string]string{
						"shipwatch": "",
					},
					Annotations: map[string]string{
						"shipwatch": "myapp",
					},
				},
			},
		},
		{
			name:         "no meta",
			answer:       false,
			instanceName: "myapp",
			input: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myappsecret",
					Namespace: metav1.NamespaceDefault,
				},
			},
		},
		{
			name:         "multiple instances",
			answer:       true,
			instanceName: "myapp",
			input: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myappsecret",
					Namespace: metav1.NamespaceDefault,
					Labels: map[string]string{
						"shipwatch": "",
					},
					Annotations: map[string]string{
						"shipwatch": "app1,myapp,otherapp",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := HasSecretMeta(test.input, test.instanceName)
			if output != test.answer {
				t.Errorf("got %t, want %t", output, test.answer)
			}
		})
	}
}
