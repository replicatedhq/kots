package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetJoinCommand(t *testing.T) {
	tests := []struct {
		name          string
		service       *corev1.Service
		secret        *corev1.Secret
		handler       http.HandlerFunc
		expectedError string
		expectedCmd   string
	}{
		{
			name: "successful join command generation",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kotsadm",
					Namespace: "kotsadm",
				},
				Spec: corev1.ServiceSpec{
					ClusterIP: "127.0.0.1",
					Ports: []corev1.ServicePort{
						{},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kotsadm-authstring",
					Namespace: "kotsadm",
				},
				Data: map[string][]byte{
					"kotsadm-authstring": []byte("test-auth-token"),
				},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case "GET":
					require.Equal(t, "/api/v1/embedded-cluster/roles", r.URL.Path)
					require.Equal(t, "test-auth-token", r.Header.Get("Authorization"))

					response := map[string]interface{}{
						"roles":              []string{"controller-role-name-normally-not-different", "worker"},
						"controllerRoleName": "test-controller-role-name",
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(response)
				case "POST":
					require.Equal(t, "/api/v1/embedded-cluster/generate-node-join-command", r.URL.Path)
					require.Equal(t, "test-auth-token", r.Header.Get("Authorization"))
					require.Equal(t, "application/json", r.Header.Get("Content-Type"))

					var requestBody struct {
						Roles []string `json:"roles"`
					}
					err := json.NewDecoder(r.Body).Decode(&requestBody)
					require.NoError(t, err)
					require.Equal(t, []string{"test-controller-role-name"}, requestBody.Roles)

					response := map[string][]string{
						"command": {"embedded-cluster", "join", "--token", "test-token"},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(response)
				}
			},
			expectedCmd: "embedded-cluster join --token test-token",
		},
		{
			name:          "missing service",
			service:       nil,
			expectedError: "unable to get kotsadm service",
		},
		{
			name: "server returns error status when fetching roles",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kotsadm",
					Namespace: "kotsadm",
				},
				Spec: corev1.ServiceSpec{
					ClusterIP: "127.0.0.1",
					Ports: []corev1.ServicePort{
						{},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kotsadm-authstring",
					Namespace: "kotsadm",
				},
				Data: map[string][]byte{
					"kotsadm-authstring": []byte("test-auth-token"),
				},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				response := map[string]string{
					"error": "internal server error",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			},
			expectedError: "failed to get roles: unexpected status code: 500",
		},
		{
			name: "server returns error status when creating token",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kotsadm",
					Namespace: "kotsadm",
				},
				Spec: corev1.ServiceSpec{
					ClusterIP: "127.0.0.1",
					Ports: []corev1.ServicePort{
						{},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kotsadm-authstring",
					Namespace: "kotsadm",
				},
				Data: map[string][]byte{
					"kotsadm-authstring": []byte("test-auth-token"),
				},
			},

			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case "GET":
					require.Equal(t, "/api/v1/embedded-cluster/roles", r.URL.Path)
					require.Equal(t, "test-auth-token", r.Header.Get("Authorization"))

					response := map[string]interface{}{
						"roles":              []string{"controller-role-name-normally-not-different", "worker"},
						"controllerRoleName": "test-controller-role-name",
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(response)
				case "POST":
					w.WriteHeader(http.StatusInternalServerError)
					response := map[string]string{
						"error": "internal server error",
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(response)
				}
			},
			expectedError: "failed to get join command: unexpected status code: 500",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a test server if we have a handler
			var server *httptest.Server
			if test.handler != nil {
				server = httptest.NewServer(test.handler)
				defer server.Close()

				// Update the service IP and port to match the test server
				serverURL, err := url.Parse(server.URL)
				require.NoError(t, err)

				host := serverURL.Hostname()
				port, err := strconv.ParseInt(serverURL.Port(), 10, 32)
				require.NoError(t, err)

				test.service.Spec.ClusterIP = host
				test.service.Spec.Ports[0].Port = int32(port)
			}

			// Create fake client with test objects
			var objects []runtime.Object
			if test.service != nil {
				objects = append(objects, test.service)
			}
			if test.secret != nil {
				objects = append(objects, test.secret)
			}
			fakeClient := fake.NewSimpleClientset(objects...)

			// Call GetJoinCommand
			cmd, err := getJoinCommandCmd(context.Background(), fakeClient, "kotsadm")

			// Verify results
			if test.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expectedCmd, cmd)
			}
		})
	}
}
