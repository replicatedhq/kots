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

	w.WriteHeader(204)
}
