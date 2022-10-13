package kurl

import (
	"reflect"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"

	"github.com/replicatedhq/kots/pkg/kurl/types"
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

func TestGetNodes(t *testing.T) {
	type args struct {
		client kubernetes.Interface
	}
	tests := []struct {
		name    string
		args    args
		want    *types.KurlNodes
		wantErr bool
	}{
		{
			name: "get nodes",
			args: args{
				client: newMockClientForListNodes(),
			},
			want: &types.KurlNodes{
				IsKurlEnabled: true,
				Nodes: []types.Node{
					{
						IsConnected: true,
						Labels:      []string{},
						CPU: types.CapacityAvailable{
							Capacity: 2, Available: 0,
						},
						Memory: types.CapacityAvailable{
							Capacity: 8, Available: 0,
						},
						Pods: types.CapacityAvailable{
							Capacity: 1000, Available: 0,
						},
					},
					{
						IsConnected: true,
						Labels:      []string{},
						CPU: types.CapacityAvailable{
							Capacity: 20, Available: 0,
						},
						Memory: types.CapacityAvailable{
							Capacity: 8, Available: 0,
						},
						Pods: types.CapacityAvailable{
							Capacity: 25, Available: 0,
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetNodes(tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetNodes() = %#v \n want %#v", got, tt.want)
			}
		})
	}
}

func newMockClientForListNodes() kubernetes.Interface {
	mockClient := fake.Clientset{}
	mockClient.AddReactor("list", "nodes", func(action core.Action) (bool, runtime.Object, error) {
		result := &corev1.NodeList{
			Items: []corev1.Node{
				{
					Status: corev1.NodeStatus{
						Capacity: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("8Gi"),
							corev1.ResourceCPU:    *resource.NewQuantity(2, resource.DecimalSI),
							corev1.ResourcePods:   resource.MustParse("1k"),
						},
					},
				},
				{
					Status: corev1.NodeStatus{
						Capacity: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("8Gi"),
							corev1.ResourceCPU:    *resource.NewQuantity(20, resource.BinarySI),
							corev1.ResourcePods:   resource.MustParse("25"),
						},
					},
				},
			},
		}
		return true, result, nil
	})
	return &mockClient
}
