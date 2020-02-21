package handlers

import (
	"net/http"
)

func CORS(w http.ResponseWriter, r *http.Request) {
	CORSHeaders(w, r)
	w.WriteHeader(http.StatusOK)
}

func CORSHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization, x-replicated-client")
}
