package handlers

import (
	"net/http"
	"net/http/httputil"
)

func NodeProxy(upstream *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
		upstream.ServeHTTP(w, r)
	}
}
