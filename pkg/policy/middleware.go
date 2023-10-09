package policy

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/rbac"
	rbactypes "github.com/replicatedhq/kots/pkg/rbac/types"
	"github.com/replicatedhq/kots/pkg/session"
	"github.com/replicatedhq/kots/pkg/store"
)

type RBACError struct {
	Resource string
}

func NewRBACError(resource string) *RBACError {
	return &RBACError{Resource: resource}
}

func (e RBACError) Abort(w http.ResponseWriter) error {
	err := fmt.Errorf("access denied to resource %s", e.Resource)
	response := ErrorResponse{Error: err.Error()}
	JSON(w, http.StatusForbidden, response)
	return err
}

type Middleware struct {
	KOTSStore store.Store
	Roles     []rbactypes.Role
}

func NewMiddleware(kotsStore store.Store, roles []rbactypes.Role) *Middleware {
	return &Middleware{
		KOTSStore: kotsStore,
		Roles:     roles,
	}
}

func (m *Middleware) EnforceAccess(p *Policy, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess := session.ContextGetSession(r)
		if sess == nil {
			logger.Error(errors.New("session empty"))
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if sess.HasRBAC { // handle pre-rbac sessions
			action, resource, err := p.execute(r, m.KOTSStore)
			if err != nil {
				logger.Error(errors.Wrapf(err, "failed to execute policy template %q", p.resource))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			rbacErr := NewRBACError(resource)

			allow, err := rbac.CheckAccess(r.Context(), m.Roles, action, resource, sess.Roles)
			if err != nil {
				logger.Error(errors.Wrapf(err, "failed to check access to resource %q", resource))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if !allow {
				logger.Error(rbacErr.Abort(w))
				return
			}
		}

		handler(w, r)
	}
}

// TODO: move everything below here to a shared package

type ErrorResponse struct {
	Error   string `json:"error"`
	Success bool   `json:"success"` // NOTE: the frontend relies on this for some routes
	Err     error  `json:"-"`
}

func NewErrorResponse(err error) ErrorResponse {
	return ErrorResponse{
		Error:   errors.Cause(err).Error(),
		Success: false,
		Err:     err,
	}
}

func JSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
