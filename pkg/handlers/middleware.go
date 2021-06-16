package handlers

import (
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/session"
	"github.com/replicatedhq/kots/pkg/store"
)

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handleOptionsRequest(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func LoggingMiddleware(next http.Handler) http.Handler {
	debug := os.Getenv("DEBUG") == "true"
	if debug {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()

			next.ServeHTTP(w, r)

			logger.Debugf(
				"request=%s method=%s duration=%s",
				r.RequestURI,
				r.Method,
				time.Since(startTime).String(),
			)
		})
	}
	return next
}

func RequireValidSessionMiddleware(kotsStore store.Store) mux.MiddlewareFunc {
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

func RequireValidSessionQuietMiddleware(kotsStore store.Store) mux.MiddlewareFunc {
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
