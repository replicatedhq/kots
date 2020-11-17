package identity

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

func NewDexProxy(address string) (func(http.ResponseWriter, *http.Request), error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	upstream := httputil.NewSingleHostReverseProxy(u)
	return handler(upstream), nil
}

func handler(upstream *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
		upstream.ServeHTTP(w, r)
	}
}
