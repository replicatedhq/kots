package handlers

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
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

func CheckAccessOrAbort(w http.ResponseWriter, r *http.Request, action, resource string) error {
	rbacErr := NewRBACError(resource)

	sess := GetSession(r)
	if sess == nil {
		return rbacErr.Abort(w)
	}

	if !sess.HasRBAC { // handle pre-rbac sessions
		return nil
	}

	// this is not very efficient to list all app slugs on each request
	appSlugs, err := store.GetStore().ListInstalledAppSlugs()
	if err != nil {
		err = errors.Wrap(err, "failed to list installed app slugs")
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	allow, err := rbac.CheckAccess(r.Context(), action, resource, sess.Roles, appSlugs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	if !allow {
		return rbacErr.Abort(w)
	}
	return nil
}
