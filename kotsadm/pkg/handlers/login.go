package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/user"
)

type LoginRequest struct {
	Password string `json:"password"`
}

type LoginResponse struct {
	Error string `json:"error,omitempty"`
	Token string `json:"token,omitempty"`
}

func Login(w http.ResponseWriter, r *http.Request) {
	loginResponse := LoginResponse{}

	loginRequest := LoginRequest{}
	if err := json.NewDecoder(r.Body).Decode(&loginRequest); err != nil {
		logger.Error(err)
		JSON(w, http.StatusBadRequest, loginResponse)
		return
	}

	foundUser, err := user.LogIn(loginRequest.Password)
	if err == user.ErrInvalidPassword {
		loginResponse.Error = "Invalid password. Please try again."
		JSON(w, http.StatusUnauthorized, loginResponse)
		return
	} else if err == user.ErrTooManyAttempts {
		loginResponse.Error = "Admin Console has been locked.  Please reset password using the \"kubectl kots reset-password\" command."
		JSON(w, http.StatusUnauthorized, loginResponse)
		return
	} else if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	createdSession, err := store.GetStore().CreateSession(foundUser)
	if err != nil {
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, loginResponse)
		return
	}

	signedJWT, err := session.SignJWT(createdSession)
	if err != nil {
		logger.Error(err)
		JSON(w, http.StatusInternalServerError, loginResponse)
		return
	}

	loginResponse.Token = fmt.Sprintf("Bearer %s", signedJWT)

	JSON(w, http.StatusOK, loginResponse)
}
