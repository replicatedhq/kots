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
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
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
	wantCommand = "sudo ./my-app join 192.168.0.100:30000 token"
	req.Equal(wantCommand, gotCommand)
}

func TestGetAllNodeIPAddresses(t *testing.T) {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	embeddedclusterv1beta1.AddToScheme(scheme)

	tests := []struct {
		name              string
		roles             []string
		kbClient          kbclient.Client
		expectedEndpoints []string
	}{
		{
			name:  "no nodes",
			roles: []string{"some-role"},
			kbClient: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&embeddedclusterv1beta1.Installation{
					ObjectMeta: metav1.ObjectMeta{
						Name: time.Now().Format("20060102150405"),
					},
					Spec: embeddedclusterv1beta1.InstallationSpec{
						BinaryName: "my-app",
						Config: &embeddedclusterv1beta1.ConfigSpec{
							Version: "v1.100.0",
							Roles: embeddedclusterv1beta1.Roles{
								Controller: embeddedclusterv1beta1.NodeRole{
									Name: "controller-role",
								},
							},
						},
					},
				},
			).Build(),
			expectedEndpoints: []string{},
		},
		{
			name:  "worker node joining cluster with 1 controller and 1 worker",
			roles: []string{"some-role"},
			kbClient: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&embeddedclusterv1beta1.Installation{
					ObjectMeta: metav1.ObjectMeta{
						Name: time.Now().Format("20060102150405"),
					},
					Spec: embeddedclusterv1beta1.InstallationSpec{
						BinaryName: "my-app",
						Config: &embeddedclusterv1beta1.ConfigSpec{
							Version: "v1.100.0",
							Roles: embeddedclusterv1beta1.Roles{
								Controller: embeddedclusterv1beta1.NodeRole{
									Name: "controller-role",
								},
							},
						},
					},
				},
				&corev1.Node{
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
				&corev1.Node{
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
			).Build(),
			expectedEndpoints: []string{"192.168.0.100:6443", "192.168.0.100:9443"},
		},
		{
			name:  "controller node joining cluster with 2 controller ready, 1 controller not ready, 1 worker ready, 1 worker not ready",
			roles: []string{"controller-role"},
			kbClient: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&embeddedclusterv1beta1.Installation{
					ObjectMeta: metav1.ObjectMeta{
						Name: time.Now().Format("20060102150405"),
					},
					Spec: embeddedclusterv1beta1.InstallationSpec{
						BinaryName: "my-app",
						Config: &embeddedclusterv1beta1.ConfigSpec{
							Version: "v1.100.0",
							Roles: embeddedclusterv1beta1.Roles{
								Controller: embeddedclusterv1beta1.NodeRole{
									Name: "controller-role",
								},
							},
						},
					},
				},
				&corev1.Node{
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
				&corev1.Node{
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
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "controller 3",
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
								Address: "192.168.0.102",
							},
						},
					},
				},
				&corev1.Node{
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
								Address: "192.168.0.103",
							},
						},
					},
				},
				&corev1.Node{
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
								Address: "192.168.0.104",
							},
						},
					},
				},
			).Build(),
			expectedEndpoints: []string{
				"192.168.0.100:6443",
				"192.168.0.100:9443",
				"192.168.0.100:2380",
				"192.168.0.100:10250",
				"192.168.0.102:6443",
				"192.168.0.102:9443",
				"192.168.0.102:2380",
				"192.168.0.102:10250",
				"192.168.0.103:10250",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			endpoints, err := GetEndpointsToCheck(context.Background(), test.kbClient, test.roles)
			req.NoError(err)
			req.Equal(test.expectedEndpoints, endpoints)
		})
	}
}
