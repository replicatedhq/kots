package handlers

import (
	"fmt"
	"net/http"

	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
	"github.com/replicatedhq/kots/kotsadm/pkg/session/types"
	"github.com/replicatedhq/kots/pkg/rbac"
)

type RBACError struct {
	Resource string
}

func NewRBACError(resource string) *RBACError {
	return &RBACError{Resource: resource}
}

type Logger struct {
}

func (l Logger) Debug(msg string, args ...interface{}) {
	line := fmt.Sprintf(msg, args...)
	logger.Debug(line)
}

func (e RBACError) Abort(w http.ResponseWriter) error {
	err := fmt.Errorf("access denied to resource %s", e.Resource)
	response := ErrorResponse{Error: err.Error()}
	JSON(w, http.StatusForbidden, response)
	return err
}

func CheckAccessOrAbort(w http.ResponseWriter, r *http.Request, resource string) error {
	rbacErr := NewRBACError(resource)

	val := r.Context().Value(sessionKey{})
	sess, ok := val.(*types.Session)
	if !ok || sess == nil {
		return rbacErr.Abort(w)
	}

	if !sess.HasRBAC { // handle pre-rbac sessions
		return nil
	}

	policies := session.RBACPoliciesFromSessionRoles(sess.Roles)

	log := Logger{}
	if rbac.CheckAccess(log, policies, resource) {
		return nil
	}

	return rbacErr.Abort(w)
}
