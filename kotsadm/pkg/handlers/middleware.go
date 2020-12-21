package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
)

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handleOptionsRequest(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireValidSessionMiddleware(kotsStore store.KOTSStore) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, err := requireValidSession(kotsStore, w, r)
			if err != nil {
				if !kotsStore.IsNotFound(err) {
					logger.Error(errors.Wrapf(err, "request %q", r.RequestURI))
				}
				return
			}

			r = session.ContextSetSession(r, sess)
			next.ServeHTTP(w, r)
		})
	}
}

func RequireValidSessionQuietMiddleware(kotsStore store.KOTSStore) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, err := requireValidSession(kotsStore, w, r)
			if err != nil {
				return
			}

			r = session.ContextSetSession(r, sess)
			next.ServeHTTP(w, r)
		})
	}
}
