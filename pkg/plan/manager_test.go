package plan

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	gwebsocket "github.com/gorilla/websocket"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/plan/types"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	"github.com/replicatedhq/kots/pkg/websocket"
	websockettypes "github.com/replicatedhq/kots/pkg/websocket/types"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var wsDialer = &gwebsocket.Dialer{
	HandshakeTimeout: 10 * time.Second,
}

func TestUpgradeECManager(t *testing.T) {
	// Create a mock store
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)

	// Test app
	versionLabel := "test-version-label"
	app := &apptypes.App{
		ID:   "test-app-id",
		Slug: "test-app",
		License: `apiVersion: kots.io/v1beta1
kind: License
metadata:
  name: test
spec:
  licenseID: test-license
  endpoint: https://replicated.app`,
	}

	p := &types.Plan{
		ID:               "test-plan",
		AppID:            app.ID,
		AppSlug:          app.Slug,
		VersionLabel:     versionLabel,
		CurrentECVersion: "1.0.0",
		NewECVersion:     "2.0.0",
	}

	type manager struct {
		nodeName string
		version  string
	}

	tests := []struct {
		name                  string
		managers              []manager
		app                   *apptypes.App
		mockStoreExpectations func()
		wantSteps             []*types.PlanStep
	}{
		{
			name: "single node upgrade",
			managers: []manager{
				{
					nodeName: "node0",
					version:  "1.0.0",
				},
			},
			app: app,
			mockStoreExpectations: func() {
				mockStore.EXPECT().GetAppFromSlug(app.Slug).Return(app, nil).Times(1)
				mockStore.EXPECT().GetPlan(app.ID, versionLabel).Return(p, nil).AnyTimes()
				mockStore.EXPECT().UpsertPlan(p).Return(nil).Times(1)
			},
			wantSteps: []*types.PlanStep{
				{
					ID:                "DYNAMIC",
					Name:              "node0 EC Manager Upgrade",
					Type:              types.StepTypeECManagerUpgrade,
					Status:            types.StepStatusPending,
					StatusDescription: "Pending EC Manager Upgrade",
					Input: types.PlanStepInputECManagerUpgrade{
						NodeName:        "node0",
						LicenseID:       "test-license",
						LicenseEndpoint: "https://replicated.app",
					},
					Owner: types.StepOwnerECManager,
				},
			},
		},
		{
			name: "2 out of 3 nodes upgrade",
			managers: []manager{
				{
					nodeName: "node0",
					version:  "1.0.0",
				},
				{
					nodeName: "node1",
					version:  "2.0.0",
				},
				{
					nodeName: "node2",
					version:  "1.0.0",
				},
			},
			app: app,
			mockStoreExpectations: func() {
				mockStore.EXPECT().GetAppFromSlug(app.Slug).Return(app, nil).Times(2)
				mockStore.EXPECT().GetPlan(app.ID, versionLabel).Return(p, nil).AnyTimes()
				mockStore.EXPECT().UpsertPlan(p).Return(nil).Times(2)
			},
			wantSteps: []*types.PlanStep{
				{
					ID:                "DYNAMIC",
					Name:              "node0 EC Manager Upgrade",
					Type:              types.StepTypeECManagerUpgrade,
					Status:            types.StepStatusPending,
					StatusDescription: "Pending EC Manager Upgrade",
					Input: types.PlanStepInputECManagerUpgrade{
						NodeName:        "node0",
						LicenseID:       "test-license",
						LicenseEndpoint: "https://replicated.app",
					},
					Owner: types.StepOwnerECManager,
				},
				{
					ID:                "DYNAMIC",
					Name:              "node2 EC Manager Upgrade",
					Type:              types.StepTypeECManagerUpgrade,
					Status:            types.StepStatusPending,
					StatusDescription: "Pending EC Manager Upgrade",
					Input: types.PlanStepInputECManagerUpgrade{
						NodeName:        "node2",
						LicenseID:       "test-license",
						LicenseEndpoint: "https://replicated.app",
					},
					Owner: types.StepOwnerECManager,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock store expectations
			tt.mockStoreExpectations()

			// Create and start test server
			ts := NewTestServer(t)
			defer ts.Close()

			// Mock EC managers
			for _, m := range tt.managers {
				go newTestECManager(ts, m.nodeName, m.version)
				ts.waitForManager(m.nodeName, m.version)
			}

			// Create fake k8s client
			scheme := runtime.NewScheme()
			err := corev1.AddToScheme(scheme)
			assert.NoError(t, err)

			objects := make([]kbclient.Object, len(tt.managers))
			for i := range tt.managers {
				objects[i] = &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: tt.managers[i].nodeName,
					},
				}
			}

			kcli := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				Build()

			// Plan the upgrade
			gotSteps, err := planECManagersUpgrade(kcli, tt.app, p.NewECVersion)
			assert.NoError(t, err)
			assert.Equal(t, len(tt.wantSteps), len(gotSteps))

			// Verify the plan steps
			for i, wantStep := range tt.wantSteps {
				wantStep.ID = gotSteps[i].ID // ID is dynamic
				assert.Equal(t, wantStep, gotSteps[i])
			}

			// Execute the plan
			p.Steps = gotSteps
			err = Execute(mockStore, p)
			assert.NoError(t, err)

			// Verify the plan status after execution
			for _, step := range p.Steps {
				assert.Equal(t, types.StepStatusComplete, step.Status)
			}

			// Verify version of EC managers
			connectedManagers := websocket.GetClients()
			assert.Equal(t, len(tt.managers), len(connectedManagers))
			for _, m := range connectedManagers {
				assert.Equal(t, p.NewECVersion, m.Version)
			}
		})
	}
}

func newTestECManager(ts *TestServer, nodeName string, version string) {
	wsURL := fmt.Sprintf("ws://%s/ec-ws?nodeName=%s&version=%s", ts.Server.Listener.Addr().String(), url.QueryEscape(nodeName), url.QueryEscape(version))
	u, err := url.Parse(wsURL)
	if err != nil {
		ts.t.Fatalf("parse websocket url: %v", err)
	}

	conn, _, err := wsDialer.Dial(u.String(), nil)
	if err != nil {
		ts.t.Fatalf("connect to websocket server: %v", err)
	}
	defer conn.Close()

Loop:
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if _, ok := err.(*gwebsocket.CloseError); ok {
				break Loop
			}
			ts.t.Fatalf("read message: %v", err)
		}

		var msg websockettypes.Message
		if err := json.Unmarshal(message, &msg); err != nil {
			ts.t.Fatalf("failed to unmarshal message: %s: %s", err, string(message))
		}
		if err := msg.Validate(); err != nil {
			ts.t.Fatalf("invalid message: %v", err)
		}

		assert.Equal(ts.t, msg.AppSlug, "test-app")
		assert.Equal(ts.t, msg.VersionLabel, "test-version-label")
		assert.NotEmpty(ts.t, msg.StepID)

		switch msg.Command {
		case websockettypes.CommandUpgradeManager:
			d := websockettypes.UpgradeManagerData{}
			if err := json.Unmarshal([]byte(msg.Data), &d); err != nil {
				ts.t.Fatalf("failed to unmarshal data: %v", err)
			}
			if err := d.Validate(); err != nil {
				ts.t.Fatalf("invalid data: %v", err)
			}

			assert.Equal(ts.t, d.LicenseID, "test-license")
			assert.Equal(ts.t, d.LicenseEndpoint, "https://replicated.app")

			// connect back with the new version
			go func() {
				time.Sleep(time.Second * 2) // simulate a restart delay
				newTestECManager(ts, nodeName, "2.0.0")
			}()

			break Loop
		default:
			ts.t.Fatalf("unknown command: %s", msg.Command)
		}
	}
}

// TestServer is a mock KOTS admin console for testing
type TestServer struct {
	Server *httptest.Server
	t      *testing.T
}

// NewTestServer creates a new test server with all the required endpoints
func NewTestServer(t *testing.T) *TestServer {
	ts := &TestServer{
		t: t,
	}

	// Reset websocket clients
	websocket.ResetClients()

	// Create the test server
	ts.Server = httptest.NewServer(http.HandlerFunc(ts.handler))

	return ts
}

func (ts *TestServer) handler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/ec-ws":
		// Handle websocket connection
		nodeName := r.URL.Query().Get("nodeName")
		assert.NotEmpty(ts.t, nodeName)

		version := r.URL.Query().Get("version")
		assert.NotEmpty(ts.t, version)

		err := websocket.Connect(w, r, nodeName, version)
		assert.NoError(ts.t, err)

	default:
		http.NotFound(w, r)
	}
}

func (ts *TestServer) waitForManager(nodeName string, version string) {
	assert.Eventually(ts.t, func() bool {
		e, ok := websocket.GetClients()[nodeName]
		return ok && e.Version == version
	}, time.Second*5, time.Millisecond*100, "Node %s did not connect", nodeName)
}

// Close shuts down the test server
func (ts *TestServer) Close() {
	ts.Server.Close()
}
