package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/identity"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/user"
	"github.com/replicatedhq/kots/pkg/util"
)

// PasswordChangeRequest - request body for the password change endpoint
type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// PasswordChangeResponse - response body for the password change endpoint
type PasswordChangeResponse struct {
	Success bool `json:"success"`
}

//  ChangePassword - change password for kots
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	identityConfig, err := identity.GetConfig(r.Context(), util.PodNamespace)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if identityConfig.Spec.Enabled && identityConfig.Spec.DisablePasswordAuth {
		err := errors.New("password authentication disabled")
		JSON(w, http.StatusForbidden, NewErrorResponse(err))
		return
	}

	var passwordChangeRequest PasswordChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&passwordChangeRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := user.ValidatePasswordInput(passwordChangeRequest.CurrentPassword, passwordChangeRequest.NewPassword); err != nil {
		JSON(w, http.StatusBadRequest, NewErrorResponse(err))
		return
	}

	// change password
	if err := user.ChangePassword(store.GetStore(), passwordChangeRequest.CurrentPassword, passwordChangeRequest.NewPassword); err != nil {
		if err == user.ErrInvalidPassword {
			JSON(w, http.StatusBadRequest, NewErrorResponse(fmt.Errorf("Your current password does not match")))
			return
		}

		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//  delete all sessions to force password change
	if err := store.GetStore().DeleteAllSessions(); err != nil {
		logger.Error(errors.Wrapf(err, "failed to delete all sessions"))
	}

	passwordChangeResponse := PasswordChangeResponse{
		Success: true,
	}
	JSON(w, http.StatusOK, passwordChangeResponse)
}
