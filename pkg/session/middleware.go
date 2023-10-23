package session

import (
	"context"
	"net/http"

	"github.com/replicatedhq/kots/pkg/session/types"
)

type sessionKey struct{}

func ContextSetSession(r *http.Request, sess *types.Session) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), sessionKey{}, sess))
}

func ContextGetSession(r *http.Request) *types.Session {
	val := r.Context().Value(sessionKey{})
	sess, _ := val.(*types.Session)
	return sess
}
