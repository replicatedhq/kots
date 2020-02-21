package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/session"
	"github.com/replicatedhq/kotsadm/pkg/user"
)

type LoginRequest struct {
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		CORSHeaders(w, r)
		return
	}

	loginRequest := LoginRequest{}
	if err := json.NewDecoder(r.Body).Decode(&loginRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	foundUser, err := user.LogIn(loginRequest.Password)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	if foundUser == nil {
		w.WriteHeader(401)
		return
	}

	createdSession, err := session.Create(foundUser)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	signedJWT, err := createdSession.SignJWT()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	loginResponse := LoginResponse{
		Token: fmt.Sprintf("Bearer %s", signedJWT),
	}

	JSON(w, 200, loginResponse)
}
