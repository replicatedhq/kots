package handlers

import (
	"net/http"

	websockettypes "github.com/replicatedhq/kots/pkg/websocket/types"
)

type DebugInfoResponse struct {
	WSClients map[string]websockettypes.WSClient `json:"wsClients"`
}

func (h *Handler) GetDebugInfo(w http.ResponseWriter, r *http.Request) {
	response := DebugInfoResponse{
		WSClients: h.WSConnectionManager.GetClients(),
	}

	JSON(w, http.StatusOK, response)
}
