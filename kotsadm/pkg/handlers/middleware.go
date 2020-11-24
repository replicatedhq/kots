package handlers

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
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

		r = r.WithContext(context.WithValue(r.Context(), sessionKey{}, sess))
		next.ServeHTTP(w, r)
	})
}

func RequireValidSessionQuietMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := requireValidSession(w, r)
		if err != nil {
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), sessionKey{}, sess))
		next.ServeHTTP(w, r)
	})
}
