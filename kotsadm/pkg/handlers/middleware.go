package handlers

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
)

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
		if err := requireValidSession(w, r); err != nil {
			logger.Error(errors.Wrapf(err, "request %q", r.RequestURI))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireValidSessionQuietMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := requireValidSession(w, r); err != nil {
			return
		}
		next.ServeHTTP(w, r)
	})
}
