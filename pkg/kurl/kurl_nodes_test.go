package kurl

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsConnected(t *testing.T) {
	tests := []struct {
		name   string
		answer bool
		node   corev1.Node
	}{
		{
			name:   "Unreachable",
			answer: false,
			node: corev1.Node{
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:       "node.kubernetes.io/unreachable",
							TimeAdded: &metav1.Time{Time: time.Now()},
						},
					},
				},
			},
		},
		{
			name:   "No taints",
			answer: true,
			node:   corev1.Node{},
		},
		{
			name:   "Not ready taint",
			answer: true,
			node: corev1.Node{
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:       "node.kubernetes.io/not-ready",
							TimeAdded: &metav1.Time{Time: time.Now()},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := isConnected(test.node)
			if output != test.answer {
				t.Errorf("got %t, want %t", output, test.answer)
			}
		})
	}
}

func TestInternalIP(t *testing.T) {
	tests := []struct {
		name   string
		answer string
		node   corev1.Node
	}{
		{
			name:   "10.128.0.42",
			answer: "10.128.0.42",
			node: corev1.Node{
				Status: corev1.NodeStatus{
					Addresses: []corev1.NodeAddress{
						{
							Type:    corev1.NodeInternalIP,
							Address: "10.128.0.42",
						},
					},
				},
			},
		},
		{
			name:   "no internal IP",
			answer: "",
			node: corev1.Node{
				Status: corev1.NodeStatus{
					Addresses: []corev1.NodeAddress{
						{
							Type:    corev1.NodeExternalIP,
							Address: "10.128.0.42",
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := internalIP(test.node)
			if output != test.answer {
				t.Errorf("got %q, want %q", output, test.answer)
			}
		})
	}
}
