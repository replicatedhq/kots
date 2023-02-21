package k8sutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakediscovery "k8s.io/client-go/discovery/fake"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func Test_IsGKEAutopilot(t *testing.T) {
	autopilotClientset := testclient.NewSimpleClientset()
	fakeDiscovery, ok := autopilotClientset.Discovery().(*fakediscovery.FakeDiscovery)
	if !ok {
		t.Error("failed to convert clientset discovery to fake discovery")
	}

	fakeDiscovery.Fake.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "auto.gke.io/v1",
			APIResources: []metav1.APIResource{
				{
					Name:    "Test API Resource",
					Group:   "auto.gke.io",
					Version: "v1",
				},
			},
		},
	}

	tests := []struct {
		name               string
		clientset          *testclient.Clientset
		isAutopilotCluster bool
	}{
		{
			name:               "is gke autopilot cluster",
			clientset:          autopilotClientset,
			isAutopilotCluster: true,
		},
		{
			name:               "not a gke autopilot cluster",
			clientset:          testclient.NewSimpleClientset(),
			isAutopilotCluster: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			isAutopilotCluster := IsGKEAutopilot(test.clientset)
			assert.Equal(t, test.isAutopilotCluster, isAutopilotCluster)
		})
	}
}
