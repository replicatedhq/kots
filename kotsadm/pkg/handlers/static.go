package handlers

import (
	"net/http"
)

type StatusNotFoundHandler struct {
}

func (h StatusNotFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "", http.StatusNotFound)
	return
}
