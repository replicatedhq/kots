package handlers

import (
	"net/http"
)

type PingResponse struct {
	Ping string `json:"ping"`
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	pingResponse := PingResponse{}
	pingResponse.Ping = "pong"
	JSON(w, http.StatusOK, pingResponse)
}
