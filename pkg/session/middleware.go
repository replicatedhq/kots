package session

import (
	"context"
	"net/http"

	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/session/types"
	"github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
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

func ContextSetLicense(r *http.Request, license *v1beta1.License) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), "kotsLicense", license))
}

func ContextGetLicense(r *http.Request) *v1beta1.License {
	val := r.Context().Value("kotsLicense")
	return val.(*v1beta1.License)
}

func ContextSetApp(r *http.Request, app *apptypes.App) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), "kotsApp", app))
}

func ContextGetApp(r *http.Request) *apptypes.App {
	val := r.Context().Value("kotsApp")
	return val.(*apptypes.App)
}
