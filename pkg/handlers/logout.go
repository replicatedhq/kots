package handlers

import (
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/session"
	"github.com/replicatedhq/kots/pkg/store"
)

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {

	signedTokenCookie, err := r.Cookie("signed-token")

	if err == http.ErrNoCookie {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	auth := signedTokenCookie.Value
	// delete cookie by setting expiration to past
	expiration := time.Now().Add(-1 * time.Hour)
	tokenCookie, err := session.GetSessionCookie(auth, expiration, r.Header.Get("Origin"))
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to delete session cookie"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, tokenCookie)

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
