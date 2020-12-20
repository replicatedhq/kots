package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/handlers"
	mock_handlers "github.com/replicatedhq/kots/kotsadm/pkg/handlers/mock"
	"github.com/replicatedhq/kots/kotsadm/pkg/policy"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
	sessiontypes "github.com/replicatedhq/kots/kotsadm/pkg/session/types"
	mock_store "github.com/replicatedhq/kots/kotsadm/pkg/store/mock"
	supportbundletypes "github.com/replicatedhq/kots/kotsadm/pkg/supportbundle/types"
	"github.com/replicatedhq/kots/pkg/rbac"
	rbactypes "github.com/replicatedhq/kots/pkg/rbac/types"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var HandlerPolicyTests = map[string][]HandlerPolicyTest{
	"GetSupportBundle": {
		{
			Vars:         map[string]string{"bundleSlug": "bundle-slug"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockKOTSStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				storeRecorder.GetSupportBundleFromSlug("bundle-slug").Return(&supportbundletypes.SupportBundle{AppID: "123"}, nil)
				storeRecorder.GetApp("123").Return(&apptypes.App{Slug: "my-app"}, nil)
				handlerRecorder.GetSupportBundle(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
	"ConfigureAppIdentityService": {
		{
			Vars:         map[string]string{"appSlug": "my-app"},
			Roles:        []rbactypes.Role{rbac.ClusterAdminRole},
			SessionRoles: []string{rbac.ClusterAdminRoleID},
			Calls: func(storeRecorder *mock_store.MockKOTSStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder) {
				handlerRecorder.ConfigureAppIdentityService(gomock.Any(), gomock.Any())
			},
			ExpectStatus: http.StatusOK,
		},
	},
}

type HandlerPolicyTest struct {
	Vars         map[string]string
	Roles        []rbactypes.Role
	SessionRoles []string
	Calls        func(storeRecorder *mock_store.MockKOTSStoreMockRecorder, handlerRecorder *mock_handlers.MockKOTSHandlerMockRecorder)
	ExpectStatus int
}

func TestHandlerPolicies(t *testing.T) {
	r := mux.NewRouter()
	// Just enough here to walk the routes
	handlers.RegisterSessionAuthRoutes(r, nil, &handlers.Handler{}, policy.NewMiddleware(nil, nil))
	r.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		name := route.GetName()
		methods, _ := route.GetMethods()
		pathTemplate, _ := route.GetPathTemplate()
		if name == "" {
			t.Errorf("route %s %s: name required", methods, pathTemplate)
			return nil
		}

		tests, ok := HandlerPolicyTests[name]
		if !ok {
			t.Errorf("route %s %s: tests required", methods, pathTemplate)
			return nil
		}

		for _, method := range methods {
			for _, test := range tests {
				pairs := []string{}
				for key, val := range test.Vars {
					pairs = append(pairs, key, val)
				}
				path, err := route.URLPath(pairs...)
				require.NoError(t, err)

				name := fmt.Sprintf("%s %s %d", method, path, test.ExpectStatus)
				t.Run(name, func(t *testing.T) {
					ctrl := gomock.NewController(t)
					defer ctrl.Finish()

					kotsStoreMock := mock_store.NewMockKOTSStore(ctrl)
					kotsHandlersMock := mock_handlers.NewMockKOTSHandler(ctrl)

					middleware := policy.NewMiddleware(kotsStoreMock, test.Roles)

					r := mux.NewRouter()
					handlers.RegisterSessionAuthRoutes(r, kotsStoreMock, kotsHandlersMock, middleware)

					sess := &sessiontypes.Session{
						ID:        ksuid.New().String(),
						IssuedAt:  time.Now(),
						ExpiresAt: time.Now().Add(time.Hour),
						Roles:     test.SessionRoles,
						HasRBAC:   true,
					}
					signedJWT, err := session.SignJWT(sess)
					require.NoError(t, err)

					req := httptest.NewRequest(method, fmt.Sprintf("http://example.com%s", path), nil)
					req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", signedJWT))

					var match mux.RouteMatch
					if !route.Match(req, &match) {
						t.Fatal("path does not match")
					}

					kotsStoreMock.EXPECT().
						GetSession(sess.ID).
						Return(sess, nil)

					test.Calls(kotsStoreMock.EXPECT(), kotsHandlersMock.EXPECT())

					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)

					resp := w.Result()

					assert.Equal(t, test.ExpectStatus, resp.StatusCode)
				})
			}
		}

		return nil
	})
}
