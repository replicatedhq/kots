package embeddedcluster

import (
	"context"
	"testing"
	"time"

	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGenerateAddNodeCommand(t *testing.T) {
	util.PodNamespace = "kotsadm"
	defer func() {
		util.PodNamespace = ""
	}()

	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	embeddedclusterv1beta1.AddToScheme(scheme)

	// Create a fake clientset
	kbClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&embeddedclusterv1beta1.Installation{
			ObjectMeta: metav1.ObjectMeta{
				Name: time.Now().Format("20060102150405"),
				Labels: map[string]string{
					"replicated.com/disaster-recovery": "ec-install",
				},
			},
			Spec: embeddedclusterv1beta1.InstallationSpec{
				BinaryName: "my-app",
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "fake-node",
				Labels: map[string]string{
					"node-role.kubernetes.io/control-plane": "true",
				},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{
						Type:   corev1.NodeReady,
						Status: corev1.ConditionTrue,
					},
				},
				Addresses: []corev1.NodeAddress{
					{
						Type:    corev1.NodeInternalIP,
						Address: "192.168.0.100",
					},
				},
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "admin-console",
				Namespace: util.PodNamespace,
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name:     "http",
						Protocol: corev1.ProtocolTCP,
						Port:     80,
						NodePort: 30000,
					},
				},
			},
		},
	).Build()

	req := require.New(t)

	// Generate the add node command for online
	gotCommand, err := GenerateAddNodeCommand(context.Background(), kbClient, "token", false)
	if err != nil {
		t.Fatalf("Failed to generate add node command: %v", err)
	}

	// Verify the generated command
	wantCommand := "sudo ./my-app join 192.168.0.100:30000 token"
	req.Equal(wantCommand, gotCommand)

	// Generate the add node command for airgap
	gotCommand, err = GenerateAddNodeCommand(context.Background(), kbClient, "token", true)
	if err != nil {
		t.Fatalf("Failed to generate add node command: %v", err)
	}

	// Verify the generated command
	wantCommand = "sudo ./my-app join --airgap-bundle my-app.airgap 192.168.0.100:30000 token"
	req.Equal(wantCommand, gotCommand)
}

func TestGetAllNodeIPAddresses(t *testing.T) {

	tests := []struct {
		name                  string
		nodes                 []corev1.Node
		expectedControllerIPs []string
		expectedWorkerIPs     []string
	}{
		{
			name:                  "no nodes",
			nodes:                 []corev1.Node{},
			expectedControllerIPs: []string{},
			expectedWorkerIPs:     []string{},
		},
		{
			name: "1 controller, 1 worker",
			nodes: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "controller",
						Labels: map[string]string{
							"node-role.kubernetes.io/control-plane": "true",
						},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{
								Type:   corev1.NodeReady,
								Status: corev1.ConditionTrue,
							},
						},
						Addresses: []corev1.NodeAddress{
							{
								Type:    corev1.NodeInternalIP,
								Address: "192.168.0.100",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker",
						Labels: map[string]string{
							"node-role.kubernetes.io/control-plane": "false",
						},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{
								Type:   corev1.NodeReady,
								Status: corev1.ConditionTrue,
							},
						},
						Addresses: []corev1.NodeAddress{
							{
								Type:    corev1.NodeInternalIP,
								Address: "192.168.0.101",
							},
						},
					},
				},
			},
			expectedControllerIPs: []string{"192.168.0.100"},
			expectedWorkerIPs:     []string{"192.168.0.101"},
		},
		{
			name: "1 controller ready, 1 controller not ready, 1 worker ready, 1 worker not ready",
			nodes: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "controller 1",
						Labels: map[string]string{
							"node-role.kubernetes.io/control-plane": "true",
						},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{
								Type:   corev1.NodeReady,
								Status: corev1.ConditionTrue,
							},
						},
						Addresses: []corev1.NodeAddress{
							{
								Type:    corev1.NodeInternalIP,
								Address: "192.168.0.100",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "controller 2",
						Labels: map[string]string{
							"node-role.kubernetes.io/control-plane": "true",
						},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{
								Type:   corev1.NodeReady,
								Status: corev1.ConditionFalse,
							},
						},
						Addresses: []corev1.NodeAddress{
							{
								Type:    corev1.NodeInternalIP,
								Address: "192.168.0.101",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "worker 1",
						Labels: map[string]string{},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{
								Type:   corev1.NodeReady,
								Status: corev1.ConditionTrue,
							},
						},
						Addresses: []corev1.NodeAddress{
							{
								Type:    corev1.NodeInternalIP,
								Address: "192.168.0.102",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker 2",
						Labels: map[string]string{
							"node-role.kubernetes.io/control-plane": "false",
						},
					},
					Status: corev1.NodeStatus{
						Conditions: []corev1.NodeCondition{
							{
								Type:   corev1.NodeReady,
								Status: corev1.ConditionFalse,
							},
						},
						Addresses: []corev1.NodeAddress{
							{
								Type:    corev1.NodeInternalIP,
								Address: "192.168.0.103",
							},
						},
					},
				},
			},
			expectedControllerIPs: []string{"192.168.0.100"},
			expectedWorkerIPs:     []string{"192.168.0.102"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			// Create a fake clientset
			scheme := runtime.NewScheme()
			corev1.AddToScheme(scheme)
			kbClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(&corev1.NodeList{Items: test.nodes}).Build()

			controllerAddr, workerAddr, err := GetAllNodeIPAddresses(context.Background(), kbClient)
			req.NoError(err)
			req.Equal(test.expectedControllerIPs, controllerAddr)
			req.Equal(test.expectedWorkerIPs, workerAddr)
		})
	}
}
