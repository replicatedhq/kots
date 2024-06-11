package handlers

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
)

type paramsKey struct{}

func SetContextParams(r *http.Request, params types.ServerParams) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), paramsKey{}, params))
}

func GetContextParams(r *http.Request) types.ServerParams {
	val := r.Context().Value(paramsKey{})
	sess, _ := val.(types.ServerParams)
	return sess
}

func ParamsMiddleware(params types.ServerParams) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = SetContextParams(r, params)
			next.ServeHTTP(w, r)
		})
	}
}

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
