package handlers

import (
	"net/http"

	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/session"
	"github.com/replicatedhq/kots/pkg/store"
)

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {

	auth := r.Header.Get("authorization")
	//TODO: remove once FE no longer sends Authorization header
	if auth == "undefined" {
		auth = ""
	}

	if auth == "" {
		signedTokenCookie, err := r.Cookie("signed-token")

		if err == http.ErrNoCookie && auth == "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		auth = signedTokenCookie.Value
	}

	sess, err := session.Parse(store.GetStore(), auth)
	if err != nil {
		if store.GetStore().IsNotFound(err) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := store.GetStore().DeleteSession(sess.ID); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
