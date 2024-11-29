package handlers

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/websocket"
)

type ConnectToECWebsocketResponse struct {
	Error string `json:"error,omitempty"`
}

func (h *Handler) ConnectToECWebsocket(w http.ResponseWriter, r *http.Request) {
	response := ConnectToECWebsocketResponse{}

	nodeName := r.URL.Query().Get("nodeName")
	if nodeName == "" {
		response.Error = "missing node name"
		logger.Error(errors.New(response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	if err := websocket.Connect(w, r, nodeName); err != nil {
		response.Error = "failed to establish websocket connection"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}
}
