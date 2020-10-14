package handlers

import (
	"net/http"
)

func CORS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, DELETE, PUT")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")
	w.Header().Set("Access-Control-Expose-Headers", "content-disposition")
}

func handleOptionsRequest(w http.ResponseWriter, r *http.Request) (isOptionsRequest bool) {
	CORS(w, r)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return true
	}
	return false
}
