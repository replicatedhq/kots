package handlers

import (
	"net/http"
)

func NotImplemented(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(501)
}
