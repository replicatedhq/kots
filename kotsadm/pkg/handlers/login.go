package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	oidc "github.com/coreos/go-oidc"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/k8s"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/user"
	usertypes "github.com/replicatedhq/kots/kotsadm/pkg/user/types"
	"github.com/replicatedhq/kots/pkg/identity"
	identityclient "github.com/replicatedhq/kots/pkg/identity/client"
	ingress "github.com/replicatedhq/kots/pkg/ingress"
	"github.com/segmentio/ksuid"
	"golang.org/x/oauth2"
)

type LoginRequest struct {
	Password string `json:"password"`
}

type LoginResponse struct {
	Error string `json:"error,omitempty"`
	Token string `json:"token,omitempty"`
}

type LoginMethod string

const (
	PasswordAuth    LoginMethod = "shared-password"
	IdentityService LoginMethod = "identity-service"
)

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	identityConfig, err := identity.GetConfig(r.Context(), os.Getenv("POD_NAMESPACE"))
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

	// TODO: super user permissions
	roles := session.GetSessionRolesFromRBAC(nil, identity.DefaultGroups)

	issuedAt, expiresAt := time.Now(), time.Now().AddDate(0, 0, 14)
	createdSession, err := store.GetStore().CreateSession(foundUser, issuedAt, expiresAt, roles)
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

type OIDCLoginResponse struct {
	AuthCodeURL string `json:"authCodeURL"`
	Error       string `json:"error,omitempty"`
}

func (h *Handler) OIDCLogin(w http.ResponseWriter, r *http.Request) {
	namespace := os.Getenv("POD_NAMESPACE")

	oidcLoginResponse := OIDCLoginResponse{}

	clientset, err := k8s.Clientset()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get k8s client"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	provider, err := identity.GetKotsadmOIDCProvider(r.Context(), clientset, namespace)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get kotsadm oidc provider"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	oauth2Config, err := identity.GetKotsadmOAuth2Config(r.Context(), clientset, namespace, *provider)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get kotsadm oauth2 config"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// generate a random state
	state := ksuid.New().String()

	// save the generated state to compare on callback
	if err := identityclient.SetOIDCState(r.Context(), namespace, state); err != nil {
		logger.Error(errors.Wrap(err, "failed to set oidc state"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	authCodeURL := oauth2Config.AuthCodeURL(state)

	oidcLoginResponse.AuthCodeURL = authCodeURL

	// return a response instead of a redirect because Dex doesn't allow redirects from different origins (CORS)
	JSON(w, http.StatusOK, oidcLoginResponse)
}

func (h *Handler) OIDCLoginCallback(w http.ResponseWriter, r *http.Request) {
	namespace := os.Getenv("POD_NAMESPACE")

	clientset, err := k8s.Clientset()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get k8s client"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	provider, err := identity.GetKotsadmOIDCProvider(r.Context(), clientset, namespace)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get kotsadm oidc provider"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	oauth2Config, err := identity.GetKotsadmOAuth2Config(r.Context(), clientset, namespace, *provider)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get kotsadm oauth2 config"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	identityConfig, err := identity.GetConfig(r.Context(), namespace)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get identity config"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var token *oauth2.Token

	switch r.Method {
	case http.MethodGet:
		// Authorization redirect callback from OAuth2 auth flow.
		if errMsg := r.FormValue("error"); errMsg != "" {
			logger.Error(errors.Wrapf(err, "%s: %s", errMsg, r.FormValue("error_description")))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		code := r.FormValue("code")
		if code == "" {
			logger.Error(errors.New("no code in request"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		state := r.FormValue("state")
		foundState, err := identityclient.GetOIDCState(r.Context(), namespace, state)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to get saved oidc state"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if foundState == "" {
			logger.Error(errors.Errorf("invalid state %s", state))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := identityclient.ResetOIDCState(r.Context(), namespace, state); err != nil {
			logger.Error(errors.Wrap(err, "failed to reset oidc state"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		httpClient, err := identityclient.HTTPClient(r.Context(), namespace, *identityConfig)
		if err != nil {
			err = errors.Wrap(err, "failed to get identity http client")
			logger.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ctx := oidc.ClientContext(r.Context(), httpClient)
		token, err = oauth2Config.Exchange(ctx, code)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to exchange token"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	case http.MethodPost:
		// Form request from frontend to refresh a token.
		refresh := r.FormValue("refresh_token")
		if refresh == "" {
			logger.Error(errors.New("no refresh_token in request"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		httpClient, err := identityclient.HTTPClient(r.Context(), namespace, *identityConfig)
		if err != nil {
			err = errors.Wrap(err, "failed to get identity http client")
			logger.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ctx := oidc.ClientContext(r.Context(), httpClient)
		t := &oauth2.Token{
			RefreshToken: refresh,
			Expiry:       time.Now().Add(-time.Hour),
		}

		token, err = oauth2Config.TokenSource(ctx, t).Token()
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to get token"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	default:
		logger.Error(errors.Errorf("method not implemented: %s", r.Method))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		logger.Error(errors.New("no id_token in token response"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: oauth2Config.ClientID})
	idToken, err := verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to verify ID token"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var claims struct {
		Email    string   `json:"email"`
		Name     string   `json:"name"`
		Verified bool     `json:"email_verified"`
		Groups   []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		logger.Error(errors.Wrap(err, "error decoding ID token claims"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user := &usertypes.User{
		ID: claims.Email,
	}

	groups := identity.DefaultGroups
	if len(identityConfig.Spec.Groups) > 0 {
		groups = identityConfig.Spec.Groups
	}
	roles := session.GetSessionRolesFromRBAC(claims.Groups, groups)

	if len(roles) == 0 {
		loginResponse := LoginResponse{}
		loginResponse.Error = "user must be a part of at least 1 group with roles"
		JSON(w, http.StatusUnauthorized, loginResponse)
		return
	}

	issuedAt, expiresAt := time.Now(), time.Now().AddDate(0, 0, 14)
	createdSession, err := store.GetStore().CreateSession(user, issuedAt, expiresAt, roles) // idToken.IssuedAt, idToken.Expiry
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to create session"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	signedJWT, err := session.SignJWT(createdSession)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to sign jwt"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	responseToken := fmt.Sprintf("Bearer %s", signedJWT)

	ingressConfig, err := ingress.GetConfig(r.Context(), namespace)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get ingress config"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	redirectURL := identityConfig.Spec.AdminConsoleAddress
	if redirectURL == "" && ingressConfig.Spec.Enabled {
		redirectURL = ingress.GetAddress(ingressConfig.Spec)
	}

	expire := time.Now().Add(30 * time.Minute)
	cookie := http.Cookie{
		Name:    "token",
		Value:   responseToken,
		Expires: expire,
		Path:    "/",
	}

	if strings.HasPrefix(redirectURL, "https") {
		cookie.Secure = true
	}

	http.SetCookie(w, &cookie)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

type GetLoginInfoResponse struct {
	Method LoginMethod `json:"method"`
	Error  string      `json:"error,omitempty"`
}

func (h *Handler) GetLoginInfo(w http.ResponseWriter, r *http.Request) {
	getLoginInfoResponse := GetLoginInfoResponse{}

	identityConfig, err := identity.GetConfig(r.Context(), os.Getenv("POD_NAMESPACE"))
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get identity config"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !identityConfig.Spec.Enabled || !identityConfig.Spec.DisablePasswordAuth {
		getLoginInfoResponse.Method = PasswordAuth
		JSON(w, http.StatusOK, getLoginInfoResponse)
		return
	}

	getLoginInfoResponse.Method = IdentityService

	JSON(w, http.StatusOK, getLoginInfoResponse)
}
