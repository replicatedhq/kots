package handlers

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	sessiontypes "github.com/replicatedhq/kots/kotsadm/pkg/session/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
)

type sessionKey struct{}

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handleOptionsRequest(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireValidSessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := requireValidSession(w, r)
		if err != nil {
			if !store.GetStore().IsNotFound(err) {
				logger.Error(errors.Wrapf(err, "request %q", r.RequestURI))
			}
			return
		}

		r = SetSession(r, sess)
		next.ServeHTTP(w, r)
	})
}

func RequireValidSessionQuietMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := requireValidSession(w, r)
		if err != nil {
			return
		}

		r = SetSession(r, sess)
		next.ServeHTTP(w, r)
	})
}

func SetSession(r *http.Request, sess *sessiontypes.Session) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), sessionKey{}, sess))
}

func GetSession(r *http.Request) *sessiontypes.Session {
	val := r.Context().Value(sessionKey{})
	sess, _ := val.(*sessiontypes.Session)
	return sess
}
