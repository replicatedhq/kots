package handlers

import (
	"net/http"

	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
)

func Logout(w http.ResponseWriter, r *http.Request) {
	if handleOptionsRequest(w, r) {
		return
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		// If there is no session, this is not an error
		if err == ErrEmptySession {
			JSON(w, 204, "")
			return
		}

		logger.Error(err)
		w.WriteHeader(401)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		JSON(w, 204, "")
		return
	}

	if err := store.GetStore().DeleteSession(sess.ID); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	JSON(w, 204, "")
}
