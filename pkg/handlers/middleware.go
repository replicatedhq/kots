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

type loggingResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.StatusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handleOptionsRequest(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func DebugLoggingMiddleware(next http.Handler) http.Handler {
	if os.Getenv("DEBUG") != "true" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		lrw := NewLoggingResponseWriter(w)
		next.ServeHTTP(lrw, r)

		logger.Debugf(
			"method=%s status=%d duration=%s request=%s",
			r.Method,
			lrw.StatusCode,
			time.Since(startTime).String(),
			r.RequestURI,
		)
	})
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		lrw := NewLoggingResponseWriter(w)
		next.ServeHTTP(lrw, r)

		if os.Getenv("DEBUG") != "true" && lrw.StatusCode < http.StatusBadRequest {
			return
		}

		logger.Infof(
			"method=%s status=%d duration=%s request=%s",
			r.Method,
			lrw.StatusCode,
			time.Since(startTime).String(),
			r.RequestURI,
		)
	})
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

func RequireValidLicenseMiddleware(kotsStore store.Store) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			license, app, err := requireValidLicense(kotsStore, w, r)
			if err != nil {
				if !kotsStore.IsNotFound(err) {
					logger.Error(errors.Wrapf(err, "request %q", r.RequestURI))
				}
				return
			}

			r = session.ContextSetLicense(r, license)
			r = session.ContextSetApp(r, app)
			next.ServeHTTP(w, r)
		})
	}
}
