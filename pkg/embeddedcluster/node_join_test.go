package embeddedcluster

import (
	"context"
	"testing"

	"github.com/replicatedhq/kots/pkg/util"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGenerateAddNodeCommand(t *testing.T) {
	util.PodNamespace = "kotsadm"
	defer func() {
		util.PodNamespace = ""
	}()

	// Create a fake clientset
	clientset := fake.NewSimpleClientset(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "embedded-cluster-config",
				Namespace: "embedded-cluster",
			},
			Data: map[string]string{
				"embedded-binary-name": "my-app",
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
	)

	req := require.New(t)

	// Generate the add node command for online
	gotCommand, err := GenerateAddNodeCommand(context.Background(), clientset, "token", false)
	if err != nil {
		t.Fatalf("Failed to generate add node command: %v", err)
	}

	// Verify the generated command
	wantCommand := "sudo ./my-app join 192.168.0.100:30000 token"
	req.Equal(wantCommand, gotCommand)

	// Generate the add node command for airgap
	gotCommand, err = GenerateAddNodeCommand(context.Background(), clientset, "token", true)
	if err != nil {
		t.Fatalf("Failed to generate add node command: %v", err)
	}

	// Verify the generated command
	wantCommand = "sudo ./my-app join --airgap-bundle my-app.airgap 192.168.0.100:30000 token"
	req.Equal(wantCommand, gotCommand)
}
