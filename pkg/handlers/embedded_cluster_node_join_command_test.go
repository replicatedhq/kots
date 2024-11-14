package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/kots/pkg/handlers/kubeclient"
	"github.com/replicatedhq/kots/pkg/store"
	mockstore "github.com/replicatedhq/kots/pkg/store/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetEmbeddedClusterNodeJoinCommand(t *testing.T) {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	embeddedclusterv1beta1.AddToScheme(scheme)

	tests := []struct {
		name              string
		kbClient          kbclient.Client
		httpStatus        int
		token             string
		getRoles          func(t *testing.T, token string) ([]string, error)
		embeddedClusterID string
		expectedBody      GetEmbeddedClusterNodeJoinCommandResponse
	}{
		{
			name:              "not an embedded cluster",
			httpStatus:        http.StatusBadRequest,
			embeddedClusterID: "",
		},
		{
			name:              "store returns error",
			httpStatus:        http.StatusInternalServerError,
			embeddedClusterID: "cluster-id",
			getRoles: func(*testing.T, string) ([]string, error) {
				return nil, fmt.Errorf("some error")
			},
		},
		{
			name:              "store gets passed the provided token",
			httpStatus:        http.StatusInternalServerError,
			embeddedClusterID: "cluster-id",
			token:             "some-token",
			getRoles: func(t *testing.T, token string) ([]string, error) {
				require.Equal(t, "some-token", token)
				return nil, fmt.Errorf("some error")
			},
		},
		{
			name:              "succesful request",
			httpStatus:        http.StatusOK,
			embeddedClusterID: "cluster-id",
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
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kube-root-ca.crt",
						Namespace: "kube-system",
					},
					Data: map[string]string{"ca.crt": "some-ca-cert"},
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
								Address: "192.168.0.101",
							},
						},
					},
				},
			).Build(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			h := &Handler{
				KubeClientBuilder: &kubeclient.MockBuilder{
					Client: test.kbClient,
				},
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mockstore.NewMockStore(ctrl)
			store.SetStore(mockStore)

			mockStore.EXPECT().GetEmbeddedClusterInstallCommandRoles(test.token).AnyTimes().DoAndReturn(func(token string) ([]string, error) {
				if test.getRoles != nil {
					return test.getRoles(t, token)
				}
				return []string{"controller-role", "worker-role"}, nil
			})

			// There's an early check in the handler for the presence of `EMBEDDED_CLUSTER_ID` env var
			// so we need to set it here whenever the test requires it
			if test.embeddedClusterID != "" {
				os.Setenv("EMBEDDED_CLUSTER_ID", test.embeddedClusterID)
				defer os.Unsetenv("EMBEDDED_CLUSTER_ID")
			}

			ts := httptest.NewServer(http.HandlerFunc(h.GetEmbeddedClusterNodeJoinCommand))
			defer ts.Close()

			url := ts.URL
			// Add token query param if provided
			if test.token != "" {
				url = fmt.Sprintf("%s?token=%s", url, test.token)
			}
			response, err := http.Get(url)
			require.Nil(t, err)
			require.Equal(t, test.httpStatus, response.StatusCode)
			if response.StatusCode != http.StatusOK {
				return
			}
			var body GetEmbeddedClusterNodeJoinCommandResponse
			require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
			require.Equal(t, test.expectedBody, body)
		})
	}

}
