package kurl

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestShouldDrain(t *testing.T) {
	tr := true
	tests := []struct {
		name   string
		pod    *corev1.Pod
		expect bool
	}{
		{
			name:   "should drain by default",
			pod:    &corev1.Pod{},
			expect: true,
		},
		{
			name: "succeeded pod should drain",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kubernetes.io/config.mirror": "27fe5f7d450d76870e10e2dbd09ca566",
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodSucceeded,
				},
			},
			expect: true,
		},
		{
			name: "failed pod should drain",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							Controller: &tr,
							Kind:       "DaemonSet",
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodSucceeded,
				},
			},
			expect: true,
		},
		{
			name: "mirror pod should not drain",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kubernetes.io/config.mirror": "27fe5f7d450d76870e10e2dbd09ca566",
					},
				},
			},
			expect: false,
		},
		{
			name: "daemonset pod should not drain",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							Controller: &tr,
							Kind:       "DaemonSet",
						},
					},
				},
			},
			expect: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := shouldDrain(test.pod)

			if actual != test.expect {
				t.Errorf("Expected %t", test.expect)
			}
		})
	}
}
