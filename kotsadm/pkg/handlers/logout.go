package handlers

import (
	"net/http"

	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
)

func Logout(w http.ResponseWriter, r *http.Request) {
	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		if store.GetStore().IsNotFound(err) {
			JSON(w, http.StatusNoContent, "")
			return
		}
		if r.Header.Get("Authorization") == "" {
			JSON(w, http.StatusNoContent, "")
			return
		}
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		JSON(w, http.StatusNoContent, "")
		return
	}

	if err := store.GetStore().DeleteSession(sess.ID); err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
