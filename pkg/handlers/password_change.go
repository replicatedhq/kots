package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/identity"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/password"
	"github.com/replicatedhq/kots/pkg/store"
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
	var passwordChangeRequest PasswordChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&passwordChangeRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := password.ValidatePasswordInput(passwordChangeRequest.CurrentPassword, passwordChangeRequest.NewPassword); err != nil {
		logger.Error(err)
		JSON(w, http.StatusBadRequest, NewErrorResponse(err))
		return
	}

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

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, NewErrorResponse(err))
		return
	}

	if err := password.ValidateCurrentPassword(store.GetStore(), passwordChangeRequest.CurrentPassword); err != nil {
		logger.Error(err)
		if errors.Is(err, password.ErrCurrentPasswordDoesNotMatch) {
			JSON(w, http.StatusBadRequest, NewErrorResponse(err))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// change password
	if err := password.ChangePassword(clientset, util.PodNamespace, passwordChangeRequest.NewPassword); err != nil {
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, NewErrorResponse(err))
		return
	}

	passwordChangeResponse := PasswordChangeResponse{
		Success: true,
	}

	logger.Info("password changed successfully")
	JSON(w, http.StatusOK, passwordChangeResponse)
}
